package binlog

import (
	"fmt"
	"net"
	log "github.com/sirupsen/logrus"
	"library/buffer"
	"time"
	"sync/atomic"
)

func (server *TcpServer) Start() {
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
func (server *TcpServer) SendClientPos(conn *tcpClientNode, data string) {
	conn.send_queue <- server.pack(CMD_POS, data)
}

// 广播
func (server *TcpServer) send(cmd int, msg string){
	log.Infof("cluster服务-广播：%s", msg)
	if server.clientsCount <= 0 {
		log.Info("cluster服务-没有连接的客户端")
		return
	}
	server.lock.Lock()
	defer server.lock.Unlock()
	for _, conn := range server.clients {
		conn.send_queue <- server.pack(cmd, msg)
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
		if !node.is_connected {
			log.Info("cluster服务-clientService退出")
			return
		}
		select {
		case msg, ok := <-node.send_queue:
			if !ok {
				log.Info("cluster服务-发送消息channel通道关闭")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 1))
			size, err := (*node.conn).Write(msg)
			atomic.AddInt64(&node.send_times, int64(1))
			if (size <= 0 || err != nil) {
				atomic.AddInt64(&server.sendFailureTimes, int64(1))
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Warn("cluster服务-失败次数：", (*node.conn).RemoteAddr().String(),
					node.sendFailureTimes)
			}
		case <-(*server.ctx).Done():
			if len(node.send_queue) <= 0 {
				log.Info("cluster服务-clientService退出")
				return
			}
		}
	}
}
func (server *TcpServer) onConnect(conn *net.Conn) {
	log.Infof("cluster服务新的连接：%s", (*conn).RemoteAddr().String())
	cnode := &tcpClientNode {
		conn               : conn,
		is_connected       : true,
		send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
		sendFailureTimes   : 0,
		weight             : 0,
		connect_time       : time.Now().Unix(),
		send_times         : int64(0),
		recvBuf            : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),
	}
	go server.clientService(cnode)
	server.lock.Lock()
	server.clients = append(server.clients[:server.clientsCount], cnode)
	server.clientsCount++
	server.lock.Unlock()
	var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
	(*conn).SetReadDeadline(time.Now().Add(time.Second*3))
	for {
		buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据 memset
		for k,_:= range buf {
			buf[k] = byte(0)
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

func (server *TcpServer) onClose(conn *tcpClientNode) {
	server.lock.Lock()
	for index, client := range server.clients {
		if client.conn == conn.conn {
			client.is_connected = false
			server.clientsCount--
			log.Warnf("cluster服务客户端掉线 %s", (*client.conn).RemoteAddr().String())
			server.clients = append(server.clients[:index], server.clients[index+1:]...)
			break
		}
	}
	server.lock.Unlock()
}

func (server *TcpServer) onMessage(conn *tcpClientNode, msg []byte) {
	conn.recvBuf.Write(msg)
	for {
		clen := conn.recvBuf.Size()
		if clen < 6 {
			return
		}
		contentLen, _  := conn.recvBuf.ReadInt32()
		cmd, _         := conn.recvBuf.ReadInt16() // 2字节 command
		content, _     := conn.recvBuf.Read(contentLen-2)
		log.Debugf("cluster服务收到消息，cmd=%d, %d, %s", cmd, contentLen, string(content))

		switch cmd {
		case CMD_POS:
			log.Debugf("cluster服务-binlog写入缓存：%s", string(content))
			server.binlog.BinlogHandler.SaveBinlogPostionCache(string(content))
		case CMD_JOIN:
			// 这里需要把服务ip和端口发送过来
			log.Debugf("cluster服务-client加入集群成功%s", (*conn.conn).RemoteAddr().String())
			(*conn.conn).SetReadDeadline(time.Time{})
			conn.send_queue <- server.pack(CMD_JOIN, "ok")
			data := fmt.Sprintf("%s:%d:%d", server.binlog.BinlogHandler.lastBinFile, server.binlog.BinlogHandler.lastPos, atomic.LoadInt64(&server.binlog.BinlogHandler.EventIndex))
			server.SendClientPos(conn, data)
			log.Debugf("cluster服务-服务节点加入集群：%s", string(content))
			conn.ServiceDns = string(content)
			// todo 这里还需要缓存起来，异常恢复的时候读取这个缓存，尝试重新加入集群
			server.saveNodes()
			index := len(server.binlog.members) + 1
			server.binlog.setMember(conn.ServiceDns, false, index)
			// todo: 将新增的节点广播给所有的节点，然后节点的members新增一个
			server.send(CMD_NEW_NODE, conn.ServiceDns)
		case CMD_GET_LEADER:
			//todo: get leader ip and response
			dns, index := server.binlog.getLeader()
			data := fmt.Sprintf("%d,%s", index, dns)
			conn.send_queue <- server.pack(CMD_GET_LEADER, data)
		default:
		}
		conn.recvBuf.ResetPos()
	}
}

func (server *TcpServer) saveNodes() {
	nodes := "[\""
	for k, v := range server.clients {
		nodes += v.ServiceDns
		if k < server.clientsCount - 1 {
			nodes += "\",\""
		} else {
			nodes += "\"]";
		}
	}
	log.Debugf("cluster写入nodes：%s", nodes)
	data := []byte(nodes)
	_, err := server.cacheHandler.WriteAt(data, 0)
	if err != nil {
		log.Errorf("cluster写入nodes缓存文件错误：%+v", err)
		return
	}
}

func (server *TcpServer) Close() {
	server.wg.Wait()
	for i := 0; i < server.clientsCount; i++ {
		(*server.clients[i].conn).Close()
	}
	if server.listener != nil {
		(*server.listener).Close()
	}
	server.cacheHandler.Close()
}