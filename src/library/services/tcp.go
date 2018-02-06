package services

import (
	"fmt"
	log "github.com/jilieryuyi/logrus"
	"net"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
	"library/app"
	"encoding/json"
)

func NewTcpService(ctx *app.Context) *TcpService {
	config, _ := GetTcpConfig()
	tcp := &TcpService {
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
		ServiceIp:        config.ServiceIp,
		Agents:           make([]*tcpClientNode, 0),
	}
	tcp.Agent = newAgent(tcp)
	for _, cgroup := range config.Groups {
		flen := len(cgroup.Filter)
		tcp.groups[cgroup.Name] = &tcpGroup{
			name: cgroup.Name,
		}
		tcp.groups[cgroup.Name].nodes = nil
		tcp.groups[cgroup.Name].filter = make([]string, flen)
		tcp.groups[cgroup.Name].filter = append(tcp.groups[cgroup.Name].filter[:0], cgroup.Filter...)
	}
	return tcp
}

// 对外的广播发送接口
func (tcp *TcpService) SendAll(data map[string] interface{}) bool {
	if !tcp.enable {
		return false
	}
	log.Debugf("tcp broadcast: %+v", data)
	//tableLen := int(msg[0]) | int(msg[1] << 8)
	table    := data["table"].(string)//string(msg[2:tableLen + 2])
	tcp.lock.Lock()
	defer tcp.lock.Unlock()

	jsonData, _ := json.Marshal(data)
	log.Debugf("datalen=%d", len(jsonData))
	//send agent
	for _, agent := range tcp.Agents {
		agent.sendQueue <- tcp.pack(CMD_EVENT, string(jsonData))
	}

	for _, cgroup := range tcp.groups {
		if cgroup.nodes == nil {
			continue
		}
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
		for _, cnode := range cgroup.nodes {
			if !cnode.isConnected {
				continue
			}
			if len(cnode.sendQueue) >= cap(cnode.sendQueue) {
				log.Warnf("tcp send channel full：%s", (*cnode.conn).RemoteAddr().String())
				continue
			}
			cnode.sendQueue <- tcp.pack(CMD_EVENT, string(jsonData))
		}
	}
	return true
}


func (tcp *TcpService) SendAll2(cmd int, msg []byte) bool {
	if !tcp.enable {
		return false
	}
	log.Debugf("tcp SendAll2 broadcast: %+v", msg)
	tcp.lock.Lock()
	defer tcp.lock.Unlock()

	//send agent
	for _, agent := range tcp.Agents {
		agent.sendQueue <- tcp.pack(cmd, string(msg))
	}

	for _, cgroup := range tcp.groups {
		if cgroup.nodes == nil {
			continue
		}
		if len(cgroup.nodes) <= 0 {
			// if node count == 0 in each group
			// program will left the loop from here
			// after len(svc.groups) loops
			continue
		}
		for _, cnode := range cgroup.nodes {
			if !cnode.isConnected {
				continue
			}
			if len(cnode.sendQueue) >= cap(cnode.sendQueue) {
				log.Warnf("tcp send channel full：%s", (*cnode.conn).RemoteAddr().String())
				continue
			}
			cnode.sendQueue <- tcp.pack(cmd, string(msg))
		}
	}
	return true
}

// data pack with little endian
func (tcp *TcpService) pack(cmd int, msg string) []byte {
	m  := []byte(msg)
	l  := len(m)
	r  := make([]byte, l+6)
	cl := l + 2
	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 24)
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
	if node.isAgent {
		for index, n := range tcp.Agents {
			if n == node {
				tcp.Agents = append(tcp.Agents[:index], tcp.Agents[index+1:]...)
				break
			}
		}
		return
	}
	if node.group != "" {
		// remove node if exists
		if group, found := tcp.groups[node.group]; found {
			for index, cnode := range group.nodes {
				if cnode.conn == node.conn {
					group.nodes = append(group.nodes[:index], group.nodes[index + 1:]...)
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
			log.Info("tcp service, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("tcp service, sendQueue channel closed.")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 1))
			size, err := (*node.conn).Write(msg)
			atomic.AddInt64(&node.sendTimes, int64(1))
			if size <= 0 || err != nil {
				atomic.AddInt64(&tcp.sendFailureTimes, int64(1))
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Warnf("tcp service, %s failure times: %d", (*node.conn).RemoteAddr().String(), node.sendFailureTimes)
			}
		case <-tcp.ctx.Ctx.Done():
			if len(node.sendQueue) <= 0 {
				log.Info("tcp service, clientSendService exit.")
				return
			}
		}
	}
}

// 连接成功回调
func (tcp *TcpService) onConnect(conn net.Conn) {
	log.Debugf("tcp service, new connect: %s", conn.RemoteAddr().String())
	cnode := &tcpClientNode{
		conn:             &conn,
		isConnected:      true,
		sendQueue:        make(chan []byte, TCP_MAX_SEND_QUEUE),
		sendFailureTimes: 0,
		connectTime:      time.Now().Unix(),
		sendTimes:        int64(0),
		recvBuf:          make([]byte, TCP_RECV_DEFAULT_SIZE),
		recvBytes:        0,
		group:            "",
		isAgent:          false,
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
		case <-tcp.ctx.Ctx.Done():
			log.Debugf("tcp服务-onConnect退出")
			return
		default:
		}
	}
}

// receive a new message
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
		clen := int(node.recvBuf[0]) | int(node.recvBuf[1] << 8) | int(node.recvBuf[2] << 16) | int(node.recvBuf[3] << 24)
		//2字节 command
		cmd  := int(node.recvBuf[4]) | int(node.recvBuf[5] << 8)
		log.Debugf("receive: cmd=%d, content_len=%d", cmd, clen)
		switch cmd {
		case CMD_SET_PRO:
			log.Info("tcp service, receive register group message")
			if len(node.recvBuf) < 10 {
				return
			}
			//内容长度+4字节的前缀（存放内容长度的数值）
			name := string(node.recvBuf[10 : clen + 4])
			tcp.lock.Lock()
			group, found := tcp.groups[name]
			if !found {
				node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("tcp service, group does not exists: %s", group))
				tcp.lock.Unlock()
				return
			}
			(*node.conn).SetReadDeadline(time.Time{})
			node.sendQueue <- tcp.pack(CMD_SET_PRO, "ok")
			node.group = group.name
			group.nodes = append(group.nodes, node)
			tcp.lock.Unlock()
		case CMD_TICK:
			node.sendQueue <- tcp.pack(CMD_TICK, "ok")
			//心跳包
		case CMD_AGENT:
			tcp.lock.Lock()
			node.isAgent = true
			(*node.conn).SetReadDeadline(time.Time{})
			tcp.Agents = append(tcp.Agents, node)
			tcp.lock.Unlock()
		default:
			node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("tcp service does not support cmd: %d", cmd))
		}
		//数据移动，清除已读数据
		node.recvBuf = append(node.recvBuf[:0], node.recvBuf[clen + 4:node.recvBytes]...)
		node.recvBytes = node.recvBytes - clen - 4
	}
}

//func (tcp *TcpService) GetIpAndPort() (string, int) {
//	return tcp.ServiceIp, tcp.Port
//}

func (tcp *TcpService) Start() {
	if !tcp.enable {
		return
	}
	//go tcp.Drive.RegisterService(tcp.ServiceIp, tcp.Port)
	go func() {
		dns := fmt.Sprintf("%s:%d", tcp.Ip, tcp.Port)
		listen, err := net.Listen("tcp", dns)
		if err != nil {
			log.Errorf("tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		log.Infof("tcp service start with: %s", dns)
		for {
			conn, err := listen.Accept()
			select {
			case <-tcp.ctx.Ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Warnf("tcp service accept with error: %+v", err)
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

//func (h *TcpService) RegisterDrive(drive cluster.Cluster) {
//	h.Drive = drive
//}

func (tcp *TcpService) Reload() {
	config, err := GetTcpConfig()
	if err != nil {
		log.Errorf("tcp service reload get config with error: %+v", err)
		return
	}
	log.Debugf("tcp service reload with new config：%+v", config)
	tcp.enable = config.Enable
	// flag to mark if need restart
	restart := false
	// check if is need restart
	if tcp.Ip != config.Listen || tcp.Port != config.Port {
		log.Debugf("tcp service need to be restarted since ip address or/and port changed from %s:%d to %s:%d",
			tcp.Ip, tcp.Port, config.Listen, config.Port)
		restart = true
		// new config
		tcp.Ip = config.Listen
		tcp.Port = config.Port
		// clear counter
		tcp.recvTimes = 0
		tcp.sendTimes = 0
		tcp.sendFailureTimes = 0
		// close all connected nodes
		// remove all groups
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
		// reset tcp config form new config
		for _, ngroup := range config.Groups { // new group
			flen := len(ngroup.Filter)
			var nodes [TCP_DEFAULT_CLIENT_SIZE]*tcpClientNode
			tcp.groups[ngroup.Name] = &tcpGroup{
				name: ngroup.Name,
			}
			tcp.groups[ngroup.Name].nodes = nodes[:0]
			tcp.groups[ngroup.Name].filter = make([]string, flen)
			tcp.groups[ngroup.Name].filter = append(tcp.groups[ngroup.Name].filter[:0], ngroup.Filter...)
		}
	} else {
		// if listen ip or/and port does not change
		// 2-direction group comparision
		for name, cgroup := range tcp.groups { // current group
			found := false
			// check the current group if exists in the new config group
			for _, ngroup := range config.Groups { // new group
				if name == ngroup.Name {
					found = true
					break
				}
			}
			// if a group does not in the new config group, remove it
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
				// if exists, reset with the new config
				// replace group filters
				group, _ := config.Groups[name]
				flen := len(group.Filter)
				tcp.groups[name].filter = make([]string, flen)
				tcp.groups[name].filter = append(tcp.groups[name].filter[:0], group.Filter...)
			}
		}
		// check new group
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
	// if need restart, restart it
	if restart {
		log.Debugf("tcp service restart...")
		tcp.Close()
		tcp.Start()
	}
}

func (tcp *TcpService) AgentStart(serviceIp string, port int) {
	tcp.Agent.Start(serviceIp, port)
}

func (tcp *TcpService) AgentStop() {
	tcp.Agent.Close()
}