package binlog

import (
	"fmt"
	"net"
	log "github.com/sirupsen/logrus"
	"library/buffer"
	"time"
	"strings"
	"sync/atomic"
)

func (server *TcpServer) Start() {
	go server.keepalive()
	go func() {
		//建立socket，监听端口
		dns := fmt.Sprintf("%s:%d", server.listen, server.port)
		listen, err := net.Listen("tcp", dns)
		if err != nil {
			log.Panicf("cluster服务错误：%+v", err)
			return
		}
		server.listener = &listen
		log.Debugf("cluster服务%s等待新的连接...", dns)
		for {
			conn, err := listen.Accept()
			select {
				case <-(*server.ctx).Done():
					return
				default:
			}
			if err != nil {
				log.Errorf("cluster服务accept错误：%+v", err)
				continue
			}
			go server.onConnect(&conn)
		}
	} ()
}

// 同步读取binlog的游标信息（当前读取到哪里了）
func (server *TcpServer) SendPos(data string) {
	server.send(CMD_POS, data)
}
func (server *TcpServer) SendClientPos(node *tcpClientNode, data string) {
	node.sendQueue <- server.pack(CMD_POS, data)
}

// 广播
func (server *TcpServer) send(cmd int, msg string){
	log.Debugf("cluster server sending broadcast: %s", msg)
	server.lock.Lock()
	defer server.lock.Unlock()
	cc := len(server.clients)
	if cc <= 0 {
		log.Debugf("no follower found, cluster server send broadcast aborted.")
		return
	}
	for _, cnode := range server.clients {
		cnode.sendQueue <- server.pack(cmd, msg)
	}
}

func (tcp *TcpServer) pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m)
	r := make([]byte, l + 6)
	cl := l + 2
	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 32)
	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], m)
	return r
}

func (server *TcpServer) clientService(node *tcpClientNode) {
	server.wg.Add(1)
	defer server.wg.Done()
	for {
		if !node.isConnected {
			log.Info("cluster服务-clientService退出")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("cluster服务-发送消息channel通道关闭")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 1))
			size, err := (*node.conn).Write(msg)
			atomic.AddInt64(&node.sendTimes, int64(1))
			if (size <= 0 || err != nil) {
				atomic.AddInt64(&server.sendFailureTimes, int64(1))
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Warn("cluster服务-失败次数：", (*node.conn).RemoteAddr().String(),
					node.sendFailureTimes)
			}
		case <-(*server.ctx).Done():
			if len(node.sendQueue) <= 0 {
				log.Info("cluster服务-clientService退出")
				return
			}
		}
	}
}

func (server *TcpServer) onConnect(conn *net.Conn) {
	log.Infof("cluster服务新的连接：%s", (*conn).RemoteAddr().String())
	cnode := &tcpClientNode {
		conn:             conn,
		isConnected:      true,
		sendQueue:        make(chan []byte, TCP_MAX_SEND_QUEUE),
		sendFailureTimes: 0,
		connectTime:      time.Now().Unix(),
		sendTimes:        int64(0),
		recvBuf:          buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),
	}
	go server.clientService(cnode)

	server.lock.Lock()
	cc := len(server.clients)
	server.clients = append(server.clients[:cc], cnode)
	server.lock.Unlock()

	var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
	(*conn).SetReadDeadline(time.Now().Add(time.Second*3))
	for {
		buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据 memset
		for i := range buf {
			buf[i] = byte(0)
		}
		size, err := (*conn).Read(buf)
		if err != nil {
			log.Errorf("cluster服务连接发生错误: %s, %v", (*conn).RemoteAddr().String(), err)
			server.onClose(cnode)
			(*conn).Close()
			return
		}
		log.Debugf("cluster服务收到消息 %d 字节：%d %s", size, buf[:size], string(buf[:size]))
		server.onMessage(cnode, buf[:size])
	}
}

func (server *TcpServer) onClose(node *tcpClientNode) {
	server.lock.Lock()
	for index, cnode := range server.clients {
		if cnode.conn == node.conn {
			cnode.isConnected = false
			log.Warnf("cluster服务客户端掉线 %s", (*cnode.conn).RemoteAddr().String())
			server.clients = append(server.clients[:index], server.clients[index+1:]...)
			break
		}
	}
	server.lock.Unlock()
	server.binlog.setStatus(node.ServiceDns, MEMBER_STATUS_LEAVE)
}

func (server *TcpServer) keepalive() {
	for {
		if server.clients == nil {
			time.Sleep(time.Second * 5)
			continue
		}
		for _, cnode := range server.clients {
			cnode.sendQueue <- server.pack(CMD_KEEPALIVE, "")
		}
		time.Sleep(time.Second * 5)
	}
}

func (server *TcpServer) onMessage(node *tcpClientNode, msg []byte) {
	node.recvBuf.Write(msg)
	for {
		size := node.recvBuf.Size()
		if size < 6 {
			return
		}
		clen, _    := node.recvBuf.ReadInt32()
		cmd, _     := node.recvBuf.ReadInt16() // 2字节 command
		content, _ := node.recvBuf.Read(clen-2)
		log.Debugf("cluster服务收到消息，cmd=%d, %d, %s", cmd, clen, string(content))

		switch cmd {
		case CMD_POS:
			log.Debugf("cluster服务-binlog写入缓存：%s", string(content))
			server.binlog.BinlogHandler.SaveBinlogPostionCache(string(content))
		case CMD_JOIN:
			// 这里需要把服务ip和端口发送过来
			log.Debugf("cluster服务-client加入集群成功%s", (*node.conn).RemoteAddr().String())
			(*node.conn).SetReadDeadline(time.Time{})
			node.sendQueue <- server.pack(CMD_JOIN, "ok")
			data := fmt.Sprintf("%s:%d:%d",
				server.binlog.BinlogHandler.lastBinFile,
				server.binlog.BinlogHandler.lastPos,
				atomic.LoadInt64(&server.binlog.BinlogHandler.EventIndex))
			server.SendClientPos(node, data)
			log.Debugf("cluster服务-服务节点加入集群：%s", string(content))
			node.ServiceDns = string(content)
			// todo 这里还需要缓存起来，异常恢复的时候读取这个缓存，尝试重新加入集群
			server.saveNodes()
			//index := len(server.binlog.members) + 1
			index := server.binlog.setMember(node.ServiceDns, false, 0)
			// todo: 将新增的节点广播给所有的节点，然后节点的members新增一个

			r := make([]byte, len(node.ServiceDns) + 2)
			r[0] = byte(index)
			r[1] = byte(index >> 8)
			copy(r[2:], node.ServiceDns)

			server.send(CMD_NEW_NODE, string(r))

			//todo 这里还需要做一件事情就是把node列表同步过去
			for dns, member := range server.binlog.members {
				if dns != node.ServiceDns {
					res := make([]byte, len(dns) + 4)
					isleader := 0
					if member.isLeader {
						isleader = 1
					}
					res[0] = byte(member.index)
					res[1] = byte(member.index >> 8)
					res[2] = byte(isleader)
					res[3] = byte(isleader >> 8)
					copy(res[4:], dns)
					server.send(CMD_NODE_SYNC, string(res))
				}
			}

		case CMD_GET_LEADER:
			//todo: get leader ip and response
			dns, index := server.binlog.getLeader()
			log.Debugf("cmd get leader is: %d, %s", index, dns)
			r := make([]byte, len(dns) + 2)
			r[0] = byte(index)
			r[1] = byte(index >> 8)
			copy(r[2:], dns)
			node.sendQueue <- server.pack(CMD_GET_LEADER, string(r))
		case CMD_CLOSE_CONFIRM:
			log.Debugf("receive close confirm from: %s", (*node.conn).RemoteAddr().String())
			server.Client.selectLeader()
			node.sendQueue <- server.pack(CMD_CLOSE_CONFIRM, "")
		case CMD_LEADER_CHANGE:
			log.Debugf("receive leader change from: %s", (*node.conn).RemoteAddr().String())
			leaderDsn := string(content)
			server.Client.ConnectTo(leaderDsn)
			node.sendQueue <- server.pack(CMD_LEADER_CHANGE, "")
		default:
		}
		node.recvBuf.ResetPos()
	}
}

func (server *TcpServer) saveNodes() {
	nodes := make([]string, len(server.clients))

	for index, cnode := range server.clients {
		nodes[index] = cnode.ServiceDns
	}
	data := []byte(fmt.Sprintf("[\"%s\"]", strings.Join(nodes, "\",\"")))
	_, err := server.cacheHandler.WriteAt(data, 0)
	if err != nil {
		log.Errorf("an error occurred when attempt to write node list file, error: %+v", err)
		return
	}
}

func (server *TcpServer) Close() {
	server.wg.Wait()
	for _, cnode := range server.clients {
		(*cnode.conn).Close()
	}
	if server.listener != nil {
		(*server.listener).Close()
	}
	server.cacheHandler.Close()
}

