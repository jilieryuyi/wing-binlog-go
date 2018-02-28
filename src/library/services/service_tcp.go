package services

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
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
		sendTimes:        0,
		sendFailureTimes: 0,
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		ServiceIp:        config.ServiceIp,
		agents:           nil,
		status:           serviceEnable | agentStatusOffline | agentStatusDisconnect,
		token:            app.GetKey(app.CachePath + "/token"),
	}
	for _, group := range config.Groups{
		tcp.groups[group.Name] = newTcpGroup(group)
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
	// pack data
	packData := pack(CMD_EVENT, data)
	//send to agents
	tcp.agents.send(packData)
	// send to all groups
	for _, group := range tcp.groups {
		if !group.match(table) {
			continue
		}
		group.send(packData)
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
	tcp.agents.send(msg)
	for _, group := range tcp.groups {
		group.send(msg)
	}
	return true
}

func (tcp *TcpService) onClose(node *tcpClientNode) {
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	node.close()
	for {
		if node.status & tcpNodeIsAgent > 0 {
			tcp.agents.remove(node)
			break
		}
		if node.status & tcpNodeIsNormal > 0 {
			if group, found := tcp.groups[node.group]; found {
				group.remove(node)
			}
			break
		}
		break
	}
}

func (tcp *TcpService) onSetPro(node *tcpClientNode, data []byte) {
	// 客户端角色分为三种
	// 一种是普通的客户端
	// 一种是用来控制进程的客户端
	// 还有另外一种就是agent代理客户端
	flag    := data[0]
	content := string(data[1:])
	switch flag {
	//set pro add to group
	case FlagSetPro:
		group, found := tcp.groups[content]
		if !found || content == "" {
			node.send(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service, group does not exists: %s", content))))
			node.close()
			return
		}
		node.setReadDeadline(time.Time{})
		node.send(packDataSetPro)
		node.setGroup(content)
		group.append(node)
		go node.asyncSendService()
	case FlagControl:
		if tcp.token != content {
			node.send(packDataTokenError)
			node.close()
			log.Warnf("token error")
			return
		}
		node.setReadDeadline(time.Time{})
		node.send(packDataSetPro)
		node.changNodeType(tcpNodeIsControl)
		go node.asyncSendService()
	case FlagAgent:
		node.setReadDeadline(time.Time{})
		node.send(packDataSetPro)
		node.changNodeType(tcpNodeIsAgent)
		tcp.agents.append(node)
		go node.asyncSendService()
	case FlagPing:
		log.Debugf("receive ping data")
		node.send(packDataSetPro)
		node.close()
	default:
		node.close()
	}
}

func (tcp *TcpService) onConnect(conn *net.Conn) {
	node := newNode(tcp.ctx, conn)
	node.setReadDeadline(time.Now().Add(time.Second * 3))
	var readBuffer [tcpDefaultReadBufferSize]byte
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	for {
		size, err := (*conn).Read(readBuffer[0:])
		if err != nil {
			if err != io.EOF {
				log.Warnf("tcp node %s disconnect with error: %v", (*conn).RemoteAddr().String(), err)
			} else {
				log.Debugf("tcp node %s disconnect with error: %v", (*conn).RemoteAddr().String(), err)
			}
			tcp.onClose(node)
			return
		}
		//log.Debugf("tcp receive: %v", readBuffer[:size])
		tcp.onMessage(node, readBuffer[:size])
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
		content := node.recvBuf[6 : clen + 4]
		switch cmd {
		case CMD_SET_PRO:
			//log.Debugf("set pro")
			tcp.onSetPro(node, content)
		case CMD_TICK:
			node.asyncSend(packDataTickOk)
		case CMD_STOP:
			//log.Debug("get stop cmd, app will stop later")
			tcp.ctx.Stop()
		case CMD_RELOAD:
			//log.Debugf("receive reload cmd：%s", string(content))
			tcp.ctx.Reload(string(content))
		case CMD_SHOW_MEMBERS:
			//log.Debugf("show members")
			tcp.ctx.ShowMembersChan <- struct{}{}
			select {
				case members, ok := <- tcp.ctx.ShowMembersRes:
					if ok && members != "" {
						node.send(pack(CMD_SHOW_MEMBERS, []byte(members)))
					}
				case <-time.After(time.Second * 30):
					node.send([]byte("get members timeout"))
			}
		default:
			node.asyncSend(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service does not support cmd: %d", cmd))))
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
			go tcp.onConnect(&conn)
		}
	}()
}

func (tcp *TcpService) Close() {
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	for _, group := range tcp.groups {
		group.close()
	}
	for _, agent := range tcp.agents {
		agent.close()
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
		//tcp.recvTimes = 0
		tcp.sendTimes = 0
		tcp.sendFailureTimes = 0
		// close all connected nodes
		// remove all groups
		for name, group := range tcp.groups {
			for _, node := range group.nodes {
				log.Debugf("closing service：%s", (*node.conn).RemoteAddr().String())
				node.close()
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
					node.close()
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
	tcp.agents.send(packData)
}