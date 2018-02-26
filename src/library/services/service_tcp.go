package services

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"library/app"
	"io"
)

func NewTcpService(ctx *app.Context) *TcpService {
	config, _ := GetTcpConfig()
	if !config.Enable{
		return &TcpService{status: serviceDisable}
	}
	tcp := &TcpService{
		Ip:               config.Listen,
		Port:             config.Port,
		lock:             new(sync.Mutex),
		groups:           make(map[string]*tcpGroup),
		recvTimes:        0,
		sendTimes:        0,
		sendFailureTimes: 0,
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		ServiceIp:        config.ServiceIp,
		Agents:           make([]*tcpClientNode, 0),
		status:           serviceEnable | agentStatusOffline | agentStatusDisconnect,
		token:            app.GetKey(app.CachePath + "/token"),
	}
	for _, group := range config.Groups{
		tcp.groups[group.Name] = &tcpGroup{
			name: group.Name,
			filter: group.Filter,
			nodes: nil,
		}
	}
	go tcp.agentKeepalive()
	return tcp
}

// send event data to all connects client
func (tcp *TcpService) SendAll(table string, data []byte) bool {
	if tcp.status & serviceDisable > 0 {
		return false
	}
	log.Debugf("tcp SendAll: %s, %+v", table, string(data))
	packData := pack(CMD_EVENT, data)
	for _, agent := range tcp.Agents {
		if agent.status & tcpNodeOnline > 0 {
			agent.sendQueue <- packData
		}
	}
	for _, group := range tcp.groups {
		if group.nodes == nil || len(group.nodes) <= 0 ||
			!matchFilters(group.filter, table) {
			continue
		}
		for _, node := range group.nodes {
			if node.status & tcpNodeOffline > 0 {
				continue
			}
			for {
				// if cache is full, try to wait it
				if len(node.sendQueue) < cap(node.sendQueue) {
					break
				}
				log.Warnf("cache full, try wait")
			}
			node.sendQueue <- packData
		}
	}
	return true
}

// send raw bytes data to all connects client
// msg is the pack frame form func: pack
func (tcp *TcpService) sendRaw(msg []byte) bool {
	if tcp.status & serviceDisable > 0 {
		return false
	}
	log.Debugf("tcp sendRaw: %+v", msg)
	for _, agent := range tcp.Agents {
		agent.sendQueue <- msg
	}
	for _, group := range tcp.groups {
		if group.nodes == nil || len(group.nodes) <= 0 {
			continue
		}
		for _, node := range group.nodes {
			if node.status & tcpNodeOffline > 0  {
				continue
			}
			for {
				if len(node.sendQueue) < cap(node.sendQueue) {
					break
				}
				log.Warnf("cache full, try wait")
			}
			node.sendQueue <- msg
		}
	}
	return true
}

func (tcp *TcpService) onClose(node *tcpClientNode) {
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if node.status & tcpNodeOnline > 0 {
		close(node.sendQueue)
	}
	if node.status & tcpNodeOnline > 0 {
		node.status ^= tcpNodeOnline
		node.status |= tcpNodeOffline
	}
	if node.status & tcpNodeIsAgent > 0 {
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
		if node.status & tcpNodeOffline > 0 {
			log.Info("tcp service, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("tcp service, sendQueue channel closed.")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 3))
			size, err := (*node.conn).Write(msg)
			atomic.AddInt64(&node.sendTimes, int64(1))
			if size <= 0 || err != nil {
				atomic.AddInt64(&tcp.sendFailureTimes, int64(1))
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Errorf("tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
			}
		case <-tcp.ctx.Ctx.Done():
			if len(node.sendQueue) <= 0 {
				log.Info("tcp service, clientSendService exit.")
				return
			}
		}
	}
}

func (tcp *TcpService) onConnect(conn net.Conn) {
	cnode := &tcpClientNode{
		conn:             &conn,
		sendQueue:        make(chan []byte, tcpMaxSendQueue),
		sendFailureTimes: 0,
		connectTime:      time.Now().Unix(),
		sendTimes:        int64(0),
		recvBuf:          make([]byte, 0),
		group:            "",
		status:           tcpNodeOnline|tcpNodeIsNotAgent,
	}
	go tcp.clientSendService(cnode)
	var readBuffer [tcpDefaultReadBufferSize]byte
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))
	for {
		if cnode.status & tcpNodeOffline > 0 {
			return
		}
		size, err := conn.Read(readBuffer[0:])
		if err != nil {
			if err != io.EOF {
				log.Warnf("tcp node %s disconnect with error: %v", conn.RemoteAddr().String(), err)
			} else {
				log.Debugf("tcp node %s disconnect with error: %v", conn.RemoteAddr().String(), err)
			}
			tcp.onClose(cnode)
			conn.Close()
			return
		}
		atomic.AddInt64(&tcp.recvTimes, int64(1))
		tcp.onMessage(cnode, readBuffer[:size])
		select {
			case <-tcp.ctx.Ctx.Done():
				log.Debugf("tcp onConnect exit")
				return
			default:
		}
	}
}

// receive a new message
func (tcp *TcpService) onMessage(node *tcpClientNode, msg []byte) {
	node.recvBuf = append(node.recvBuf, msg...)
	//log.Debugf("tcp node.recvBuf len: %d, cap: %d", len(node.recvBuf), cap(node.recvBuf))
	for {
		size := len(node.recvBuf)
		if size < 6 {
			return
		}
		//log.Debugf("buffer: %v", node.recvBuf)
		clen := int(node.recvBuf[0]) | int(node.recvBuf[1]) << 8 |
			int(node.recvBuf[2]) << 16 | int(node.recvBuf[3]) << 24
		if len(node.recvBuf) < 	clen + 4 {
			return
		}
		//2字节 command
		cmd  := int(node.recvBuf[4]) | int(node.recvBuf[5]) << 8
		if !hasCmd(cmd) {
			log.Errorf("cmd %d does not exists, data: %v", cmd, node.recvBuf)
			node.recvBuf = make([]byte, 0)
			return
		}
		//log.Debugf("receive: cmd=%d, content_len=%d", cmd, clen)
		content := string(node.recvBuf[6 : clen + 4])
		switch cmd {
		case CMD_SET_PRO:
			log.Info("tcp service, receive register group message")
			if len(node.recvBuf) < 7 {
				return
			}
			//内容长度+4字节的前缀（存放内容长度的数值）
			name := content//string(node.recvBuf[6 : clen + 4])
			log.Debugf("add to group: %s", name)
			tcp.lock.Lock()
			group, found := tcp.groups[name]
			if !found {
				node.sendQueue <- pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service, group does not exists: %s", group)))
				tcp.lock.Unlock()
				return
			}
			(*node.conn).SetReadDeadline(time.Time{})
			node.sendQueue <- packDataSetPro
			node.group = group.name
			group.nodes = append(group.nodes, node)
			tcp.lock.Unlock()
		case CMD_TICK:
			//log.Debugf("cmd tick")
			node.sendQueue <- packDataTickOk
		case CMD_AGENT:
			tcp.lock.Lock()
			if node.status & tcpNodeIsNotAgent > 0 {
				node.status ^= tcpNodeIsNotAgent
				node.status |= tcpNodeIsAgent
			}
			(*node.conn).SetReadDeadline(time.Time{})
			tcp.Agents = append(tcp.Agents, node)
			tcp.lock.Unlock()
		case CMD_AUTH:
			if content == tcp.token {
				(*node.conn).SetReadDeadline(time.Time{})
			} else {
				(*node.conn).Write(packDataTokenError)
				(*node.conn).Close()
				log.Warnf("auth error: %s", content)
			}
		case CMD_STOP:
			log.Debug("get stop cmd, app will stop later")
			tcp.ctx.CancelChan <- struct{}{}
		case CMD_RELOAD:
			content := string(node.recvBuf[6 : clen + 4])
			log.Debugf("receive reload cmd：%s", string(content))
			tcp.ctx.ReloadChan <- string(content)
		case CMD_SHOW_MEMBERS:
			tcp.ctx.ShowMembersChan <- struct{}{}
			select {
				case members, ok := <- tcp.ctx.ShowMembersRes:
					if ok && members != "" {
						(*node.conn).Write(pack(CMD_SHOW_MEMBERS, []byte(members)))
					}
				case <-time.After(time.Second * 30):
					(*node.conn).Write([]byte("get members timeout"))
			}
		default:
			node.sendQueue <- pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service does not support cmd: %d", cmd)))
			node.recvBuf = make([]byte, 0)
			return
		}
		//数据移动，清除已读数据
		node.recvBuf = append(node.recvBuf[:0], node.recvBuf[clen + 4:]...)
		//log.Debugf("tcp node.recvBuf len: %d, cap: %d", len(node.recvBuf), cap(node.recvBuf))
	}
}

func (tcp *TcpService) Start() {
	if tcp.status & serviceDisable > 0 {
		return
	}
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
			if cnode.status & tcpNodeOnline > 0 {
				cnode.status ^= tcpNodeOnline
				cnode.status |= tcpNodeOffline
			}
		}
	}
	log.Debugf("tcp service closed.")
}

func (tcp *TcpService) Reload() {
	config, err := GetTcpConfig()
	if err != nil {
		log.Errorf("tcp service reload get config with error: %+v", err)
		return
	}
	log.Debugf("tcp service reload with new config：%+v", config)
	if config.Enable && tcp.status & serviceDisable > 0 {
		tcp.status ^= serviceDisable
		tcp.status |= serviceEnable
	}
	if !config.Enable && tcp.status & serviceEnable > 0 {
		tcp.status ^= serviceEnable
		tcp.status |= serviceDisable
	}
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
		for name, group := range tcp.groups {
			for _, node := range group.nodes {
				log.Debugf("closing service：%s", (*node.conn).RemoteAddr().String())
				if node.status & tcpNodeOnline > 0 {
					node.status ^= tcpNodeOnline
					node.status |= tcpNodeOffline
				}
				close(node.sendQueue)
				(*node.conn).Close()
			}
			log.Debugf("removing groups：%s", name)
			delete(tcp.groups, name)
		}
		// reset tcp config form new config
		for _, group := range config.Groups { // new group
			tcp.groups[group.Name] = &tcpGroup{
				name: group.Name,
				filter: group.Filter,
				nodes: nil,
			}
		}
	} else {
		// if listen ip or/and port does not change
		// 2-direction group comparision
		for name, group := range tcp.groups { // current group
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
				for _, node := range group.nodes {
					log.Debugf("closing connection: %s", (*node.conn).RemoteAddr().String())
					close(node.sendQueue)
					//cnode.isConnected = false
					if node.status & tcpNodeOnline > 0 {
						node.status ^= tcpNodeOnline
						node.status |= tcpNodeOffline
					}
					(*node.conn).Close()
				}
				delete(tcp.groups, name)
			} else {
				// if exists, reset with the new config
				// replace group filters
				group, _ := config.Groups[name]
				//flen := len(group.Filter)
				tcp.groups[name].filter = nil//make([]string, flen)
				tcp.groups[name].filter = append(tcp.groups[name].filter, group.Filter...)
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
			if found {
				continue
			}
			// add it if new group found
			log.Debugf("new group: %s", ngroup.Name)
			tcp.groups[ngroup.Name] = &tcpGroup{
				name: ngroup.Name,
				nodes:nil,
				filter:ngroup.Filter,
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

func (tcp *TcpService) SendPos(data []byte) {
	packData := pack(CMD_POS, data)
	for _, agent := range tcp.Agents {
		if agent.status & tcpNodeOnline > 0 {
			for {
				if len(agent.sendQueue) < cap(agent.sendQueue) {
					break
				}
				log.Warnf("cache full, try wait")
			}
			agent.sendQueue <- packData
		}
	}
}