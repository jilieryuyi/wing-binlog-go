package services

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

func NewTcpService(ctx *context.Context) *TcpService {
	config, _ := getTcpConfig()
	tcp := &TcpService{
		Ip:               config.Listen,
		Port:             config.Port,
		lock:             new(sync.Mutex),
		groups:           make(map[string]*tcpGroup),
		recvTimes:        0,
		sendTimes:        0,
		sendFailureTimes: 0,
		enable:           config.Enable,
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
	}
	for _, cgroup := range config.Groups {
		flen := len(cgroup.Filter)
		var nodes [TCP_DEFAULT_CLIENT_SIZE]*tcpClientNode
		tcp.groups[cgroup.Name] = &tcpGroup{
			name: cgroup.Name,
			mode: cgroup.Mode,
		}
		tcp.groups[cgroup.Name].nodes = nodes[:0]
		tcp.groups[cgroup.Name].filter = make([]string, flen)
		tcp.groups[cgroup.Name].filter = append(tcp.groups[cgroup.Name].filter[:0], cgroup.Filter...)
	}
	return tcp
}

// 对外的广播发送接口
func (tcp *TcpService) SendAll(msg []byte) bool {
	if !tcp.enable {
		return false
	}
	log.Info("tcp服务-广播：", string(msg))
	table_len := int(msg[0]) + int(msg[1]<<8)
	table := string(msg[2 : table_len+2])

	tcp.lock.Lock()
	defer tcp.lock.Unlock()

	for _, cgroup := range tcp.groups {
		// 如果分组里面没有客户端连接，跳过
		if len(cgroup.nodes) <= 0 {
			// if node count == 0 in each group
			// program will left the loop from here
			// after len(svc.groups) loops
			continue
		}
		// check if the table name matches the filter
		if len(cgroup.filter) > 0 {
			found := false
			for _, f := range cgroup.filter {
				match, err := regexp.MatchString(f, table)
				if err != nil {
					continue
				}
				if match {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		mode := cgroup.mode
		// 如果不等于权重，即广播模式
		if mode != MODEL_WEIGHT {
			for _, cnode := range cgroup.nodes {
				if !cnode.isConnected {
					continue
				}
				log.Info("tcp服务-发送广播消息")
				if len(cnode.sendQueue) >= cap(cnode.sendQueue) {
					log.Warnf("tcp服务-发送缓冲区满：%s", (*cnode.conn).RemoteAddr().String())
					continue
				}
				cnode.sendQueue <- tcp.pack(CMD_EVENT, string(msg[table_len+2:])) //msg[table_len+2:]
			}
		} else {
			// 负载均衡模式
			// todo 根据已经send_times的次数负载均衡
			target := cgroup.nodes[0]
			//将发送次数/权重 作为负载基数，每次选择最小的发送
			js := float64(atomic.LoadInt64(&target.sendTimes)) / float64(target.weight)
			clen := len(cgroup.nodes)
			for i := 1; i < clen; i++ {
				stimes := atomic.LoadInt64(&cgroup.nodes[i].sendTimes)
				//conn.send_queue <- msg
				if stimes == 0 {
					//优先发送没有发过的
					target = cgroup.nodes[i]
					break
				}
				njs := float64(stimes) / float64(cgroup.nodes[i].weight)
				if njs < js {
					js = njs
					target = cgroup.nodes[i]
				}
			}
			log.Info("tcp服务-发送权重消息，", (*target.conn).RemoteAddr().String())
			if len(target.sendQueue) >= cap(target.sendQueue) {
				log.Warnf("tcp服务-发送缓冲区满：%s", (*target.conn).RemoteAddr().String())
			} else {
				target.sendQueue <- tcp.pack(CMD_EVENT, string(msg[table_len+2:]))
			}
		}
	}
	return true
}

// 数据封包
func (tcp *TcpService) pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m)
	r := make([]byte, l+6)
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

// 掉线回调
func (tcp *TcpService) onClose(node *tcpClientNode) {
	tcp.lock.Lock()
	defer tcp.lock.Unlock()

	close(node.sendQueue)
	node.isConnected = false

	if node.group != "" {
		// remove node if exists
		if group, found := tcp.groups[node.group]; found {
			for index, cnode := range group.nodes {
				if cnode.conn == node.conn {
					group.nodes = append(group.nodes[:index], group.nodes[index+1:]...)
					break
				}
			}
		}
	}
}

// 客户端服务协程，一个客户端一个
func (tcp *TcpService) clientSendService(node *tcpClientNode) {
	tcp.wg.Add(1)
	defer tcp.wg.Done()
	for {
		if !node.isConnected {
			log.Info("tcp服务-clientSendService退出")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("tcp服务-发送消息channel通道关闭")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 1))
			size, err := (*node.conn).Write(msg)
			atomic.AddInt64(&node.sendTimes, int64(1))
			if size <= 0 || err != nil {
				atomic.AddInt64(&tcp.sendFailureTimes, int64(1))
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Warn("tcp服务-失败次数：", (*node.conn).RemoteAddr().String(), node.sendFailureTimes)
			}
		case <-(*tcp.ctx).Done():
			if len(node.sendQueue) <= 0 {
				log.Info("tcp服务-clientSendService退出")
				return
			}
		}
	}
}

// 连接成功回调
func (tcp *TcpService) onConnect(conn net.Conn) {
	log.Info("tcp服务-新的连接：", conn.RemoteAddr().String())
	cnode := &tcpClientNode{
		conn:             &conn,
		isConnected:      true,
		sendQueue:        make(chan []byte, TCP_MAX_SEND_QUEUE),
		sendFailureTimes: 0,
		weight:           0,
		mode:             MODEL_BROADCAST,
		connectTime:      time.Now().Unix(),
		sendTimes:        int64(0),
		recvBuf:          make([]byte, TCP_RECV_DEFAULT_SIZE),
		recvBytes:        0,
		group:            "",
	}
	go tcp.clientSendService(cnode)
	var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))
	for {
		buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据 memset
		for i := range buf {
			buf[i] = byte(0)
		}
		size, err := conn.Read(buf)
		if err != nil {
			log.Warn("tcp服务-连接发生错误: ", conn.RemoteAddr().String(), err)
			tcp.onClose(cnode)
			conn.Close()
			return
		}
		log.Debug("tcp服务-收到消息", size, "字节：", buf[:size], string(buf))
		atomic.AddInt64(&tcp.recvTimes, int64(1))
		cnode.recvBytes += size
		tcp.onMessage(cnode, buf, size)

		select {
		case <-(*tcp.ctx).Done():
			log.Debugf("tcp服务-onConnect退出")
			return
		default:
		}
	}
}

// 收到消息回调函数
func (tcp *TcpService) onMessage(node *tcpClientNode, msg []byte, size int) {
	node.recvBuf = append(node.recvBuf[:node.recvBytes-size], msg[0:size]...)
	for {
		size := len(node.recvBuf)
		if size < 6 {
			return
		} else if size > TCP_RECV_DEFAULT_SIZE {
			// 清除所有的读缓存，防止发送的脏数据不断的累计
			node.recvBuf = make([]byte, TCP_RECV_DEFAULT_SIZE)
			log.Info("tcp服务-新建缓冲区")
			return
		}
		//4字节长度
		clen := int(node.recvBuf[0]) +
			int(node.recvBuf[1]<<8) +
			int(node.recvBuf[2]<<16) +
			int(node.recvBuf[3]<<32)
		//2字节 command
		cmd := int(node.recvBuf[4]) + int(node.recvBuf[5]<<8)
		log.Debugf("收到消息：cmd=%d, content_len=%d", cmd, clen)
		switch cmd {
		case CMD_SET_PRO:
			log.Info("tcp服务-收到注册分组消息")
			if len(node.recvBuf) < 10 {
				return
			}
			//4字节 weight
			weight := int(node.recvBuf[6]) +
				int(node.recvBuf[7]<<8) +
				int(node.recvBuf[8]<<16) +
				int(node.recvBuf[9]<<32)
			if weight < 0 || weight > 100 {
				node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的权重值：%d，请设置为0-100之间", weight))
				return
			}
			//内容长度+4字节的前缀（存放内容长度的数值）
			name := string(node.recvBuf[10 : clen+4])
			tcp.lock.Lock()
			group, found := tcp.groups[name]
			if !found {
				node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("tcp服务-组不存在：%s", group))
				tcp.lock.Unlock()
				return
			}
			(*node.conn).SetReadDeadline(time.Time{})
			node.sendQueue <- tcp.pack(CMD_SET_PRO, "ok")
			node.group = group.name
			node.mode = group.mode
			node.weight = weight
			group.nodes = append(group.nodes, node)
			if node.mode == MODEL_WEIGHT {
				//weight 合理性格式化，保证所有的weight的和是100
				all_weight := 0
				for _, cnode := range group.nodes {
					w := cnode.weight
					if w <= 0 {
						w = 100
					}
					all_weight += w
				}

				gl := len(group.nodes)
				yg := 0
				for index, cnode := range group.nodes {
					if index == gl-1 {
						cnode.weight = 100 - yg
					} else {
						cnode.weight = int(cnode.weight * 100 / all_weight)
						yg += cnode.weight
					}
				}
			}
			tcp.lock.Unlock()
		case CMD_TICK:
			node.sendQueue <- tcp.pack(CMD_TICK, "ok")
		//心跳包
		default:
			node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
		}
		//数据移动
		node.recvBuf = append(node.recvBuf[:0], node.recvBuf[clen+4:node.recvBytes]...)
		node.recvBytes = node.recvBytes - clen - 4
	}
}

func (tcp *TcpService) Start() {
	if !tcp.enable {
		return
	}
	go func() {
		//建立socket，监听端口
		dns := fmt.Sprintf("%s:%d", tcp.Ip, tcp.Port)
		listen, err := net.Listen("tcp", dns)
		if err != nil {
			log.Error("tcp服务发生错误：", err)
			return
		}
		tcp.listener = &listen
		log.Infof("tcp服务-等待新的连接...")
		for {
			conn, err := listen.Accept()
			select {
			case <-(*tcp.ctx).Done():
				return
			default:
			}
			if err != nil {
				log.Warn("tcp服务发生错误：", err)
				continue
			}
			go tcp.onConnect(conn)
		}
	}()
}

func (tcp *TcpService) Close() {
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()

	for _, cgroup := range tcp.groups {
		if len(cgroup.nodes) > 0 {
			tcp.wg.Wait()
			break
		}
	}

	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	for _, cgroup := range tcp.groups {
		for _, cnode := range cgroup.nodes {
			close(cnode.sendQueue)
			(*cnode.conn).Close()
			cnode.isConnected = false
		}
	}

	log.Debugf("tcp service closed.")
}

// 重新加载服务
func (tcp *TcpService) Reload() {
	log.Debug("tcp服务reload...")
	config, _ := getTcpConfig()
	log.Debugf("新配置：%+v", config)
	tcp.enable = config.Enable
	restart := false

	if tcp.Ip != config.Listen || tcp.Port != config.Port {
		log.Debugf("tcp service need to be restarted since ip address or/and port changed from %s:%d to %s:%d",
			tcp.Ip,
			tcp.Port,
			config.Listen,
			config.Port)
		restart = true
		tcp.Ip = config.Listen
		tcp.Port = config.Port
		tcp.recvTimes = 0
		tcp.sendTimes = 0
		tcp.sendFailureTimes = 0

		for name, cgroup := range tcp.groups {
			for _, cnode := range cgroup.nodes {
				log.Debugf("closing service：%s", (*cnode.conn).RemoteAddr().String())
				cnode.isConnected = false
				close(cnode.sendQueue)
				(*cnode.conn).Close()
			}
			log.Debugf("removing groups：%s", name)
			delete(tcp.groups, name)
		}
		for _, ngroup := range config.Groups { // new group
			flen := len(ngroup.Filter)
			var nodes [TCP_DEFAULT_CLIENT_SIZE]*tcpClientNode
			tcp.groups[ngroup.Name] = &tcpGroup{
				name: ngroup.Name,
				mode: ngroup.Mode,
			}
			tcp.groups[ngroup.Name].nodes = nodes[:0]
			tcp.groups[ngroup.Name].filter = make([]string, flen)
			tcp.groups[ngroup.Name].filter = append(tcp.groups[ngroup.Name].filter[:0], ngroup.Filter...)
		}
	} else {
		// 2-direction group comparision
		for name, cgroup := range tcp.groups { // current group
			found := false
			for _, ngroup := range config.Groups { // new group
				if name == ngroup.Name {
					found = true
					break
				}
			}
			// remove group
			if !found {
				log.Debugf("group removed: %s", name)
				for _, cnode := range cgroup.nodes {
					log.Debugf("closing connection: %s", (*cnode.conn).RemoteAddr().String())
					close(cnode.sendQueue)
					cnode.isConnected = false
					(*cnode.conn).Close()
				}
				delete(tcp.groups, name)
			} else {
				// replace group filters
				group, _ := config.Groups[name]
				flen := len(group.Filter)
				tcp.groups[name].filter = make([]string, flen)
				tcp.groups[name].filter = append(tcp.groups[name].filter[:0], group.Filter...)
				tcp.groups[name].mode = group.Mode
			}
		}

		for _, ngroup := range config.Groups { // new group
			found := false
			for name := range tcp.groups {
				if name == ngroup.Name {
					found = true
					break
				}
			}

			// add it if new group found
			if !found {
				log.Debugf("new group: %s", ngroup.Name)
				flen := len(ngroup.Filter)
				var nodes [TCP_DEFAULT_CLIENT_SIZE]*tcpClientNode
				tcp.groups[ngroup.Name] = &tcpGroup{
					name: ngroup.Name,
					mode: ngroup.Mode,
				}
				tcp.groups[ngroup.Name].nodes = nodes[:0]
				tcp.groups[ngroup.Name].filter = make([]string, flen)
				tcp.groups[ngroup.Name].filter = append(tcp.groups[ngroup.Name].filter[:0], ngroup.Filter...)
			} else {
				// do nothing, the existing group is already processed in 1st round comparision
				continue
			}
		}
	}

	if restart {
		log.Debugf("tcp服务重启...")
		tcp.Close()
		tcp.Start()
	}
}
