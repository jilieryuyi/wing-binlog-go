package agent

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"library/app"
	"io"
	"sync/atomic"
)

func NewTcpService(ctx *app.Context) *TcpService {
	tcp := &TcpService{
		Address:          ctx.AppConfig.AgentfListen,
		lock:             new(sync.Mutex),
		statusLock:       new(sync.Mutex),
		groups:           make(map[string]*tcpGroup),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		ServiceIp:        ctx.TcpConfig.ServiceIp,
		agents:           nil,
		status:           serviceEnable,
		token:            app.GetKey(app.CachePath + "/token"),
	}
	for _, group := range ctx.TcpConfig.Groups{
		tcpGroup := newTcpGroup(group)
		tcp.lock.Lock()
		tcp.groups.add(tcpGroup)
		tcp.lock.Unlock()
	}
	go tcp.agentKeepalive()
	go tcp.keepalive()
	return tcp
}

// send event data to all connects client
func (tcp *TcpService) SendAll(table string, data []byte) bool {
	tcp.statusLock.Lock()
	if tcp.status & serviceEnable <= 0 {
		tcp.statusLock.Unlock()
		return false
	}
	tcp.statusLock.Unlock()
	log.Debugf("tcp SendAll: %s, %+v", table, string(data))
	// pack data
	packData := pack(CMD_EVENT, data)
	//send to agents
	tcp.agents.asyncSend(packData)
	// send to all groups
	for _, group := range tcp.groups {
		// check if match
		if group.match(table) {
			group.asyncSend(packData)
		}
	}
	return true
}

// send raw bytes data to all connects client
// msg is the pack frame form func: pack
func (tcp *TcpService) sendRaw(msg []byte) bool {
	tcp.statusLock.Lock()
	if tcp.status & serviceEnable <= 0 {
		tcp.statusLock.Unlock()
		return false
	}
	tcp.statusLock.Unlock()
	log.Debugf("tcp sendRaw: %+v", msg)
	tcp.agents.asyncSend(msg)
	tcp.groups.asyncSend(msg)
	return true
}

func (tcp *TcpService) onClose(node *tcpClientNode) {
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	node.close()
	if node.status & tcpNodeIsAgent > 0 {
		tcp.agents.remove(node)
		return
	}
	if node.status & tcpNodeIsNormal > 0 {
		if group, found := tcp.groups[node.group]; found {
			group.remove(node)
		}
		return
	}
}

func (tcp *TcpService) onSetPro(node *tcpClientNode, groupName string) {
	group, found := tcp.groups[groupName]
	if !found || groupName == "" {
		node.send(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service, group does not exists: %s", groupName))))
		node.close()
		return
	}
	node.setReadDeadline(time.Time{})
	node.send(packDataSetPro)
	node.setGroup(groupName)
	group.append(node)
	go tcp.asyncSendService(node)
}

//func (tcp *TcpService) onControl(node *tcpClientNode, token string) {
//	if tcp.token != token {
//		node.send(packDataTokenError)
//		node.close()
//		log.Warnf("token error")
//		return
//	}
//	node.setReadDeadline(time.Time{})
//	node.send(packDataSetPro)
//	node.changNodeType(tcpNodeIsControl)
//	go tcp.asyncSendService(node)
//}

func (tcp *TcpService) onAgent(node *tcpClientNode) {
	node.setReadDeadline(time.Time{})
	node.send(packDataSetPro)
	node.changNodeType(tcpNodeIsAgent)
	tcp.agents.append(node)
	go tcp.asyncSendService(node)
}

func (tcp *TcpService) onPing(node *tcpClientNode) {
	log.Debugf("receive ping data")
	node.send(packDataSetPro)
	node.close()
}

func (tcp *TcpService) onSetProEvent(node *tcpClientNode, data []byte) {
	// 客户端角色分为三种
	// 一种是普通的客户端
	// 一种是用来控制进程的客户端
	// 还有另外一种就是agent代理客户端
	flag    := data[0]
	content := string(data[1:])
	switch flag {
	case FlagSetPro:
		tcp.onSetPro(node, content)
	//case FlagControl:
	//	tcp.onControl(node, content)
	case FlagAgent:
		tcp.onAgent(node)
	case FlagPing:
		tcp.onPing(node)
	default:
		node.close()
	}
}

func (tcp *TcpService) onShowMembersEvent(node *tcpClientNode) {
	tcp.ctx.ShowMembersChan <- struct{}{}
	select {
	case members, ok := <- tcp.ctx.ShowMembersRes:
		if ok && members != "" {
			node.send(pack(CMD_SHOW_MEMBERS, []byte(members)))
		}
	case <-time.After(time.Second * 30):
		node.send([]byte("get members timeout"))
	}
}

func (tcp *TcpService) asyncSendService(node *tcpClientNode) {
	tcp.wg.Add(1)
	defer tcp.wg.Done()
	for {
		if node.status & tcpNodeOnline <= 0 {
			log.Info("tcp node is closed, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("tcp node sendQueue is closed, sendQueue channel closed.")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 30))
			size, err := (*node.conn).Write(msg)
			if err != nil {
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Errorf("tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
				tcp.onClose(node)
				return
			}
			if size != len(msg) {
				log.Errorf("%s send not complete: %v", (*node.conn).RemoteAddr().String(), msg)
			}
		case <-node.ctx.Ctx.Done():
			log.Debugf("context is closed, wait for exit, left: %d", len(node.sendQueue))
			if len(node.sendQueue) <= 0 {
				log.Info("tcp service, clientSendService exit.")
				return
			}
		}
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
	for {
		size := len(node.recvBuf)
		if size < 6 {
			return
		}
		clen := int(node.recvBuf[0]) | int(node.recvBuf[1]) << 8 |
			int(node.recvBuf[2]) << 16 | int(node.recvBuf[3]) << 24
		if len(node.recvBuf) < 	clen + 4 {
			return
		}
		cmd  := int(node.recvBuf[4]) | int(node.recvBuf[5]) << 8
		if !hasCmd(cmd) {
			log.Errorf("cmd %d does not exists, data: %v", cmd, node.recvBuf)
			node.recvBuf = make([]byte, 0)
			return
		}
		content := node.recvBuf[6 : clen + 4]
		switch cmd {
		case CMD_SET_PRO:
			tcp.onSetProEvent(node, content)
		case CMD_TICK:
			node.asyncSend(packDataTickOk)
		//case CMD_STOP:
		//	tcp.ctx.Stop()
		//case CMD_RELOAD:
		//	tcp.ctx.Reload(string(content))
		//case CMD_SHOW_MEMBERS:
		//	tcp.onShowMembersEvent(node)
		default:
			node.asyncSend(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service does not support cmd: %d", cmd))))
			node.recvBuf = make([]byte, 0)
			return
		}
		node.recvBuf = append(node.recvBuf[:0], node.recvBuf[clen + 4:]...)
	}
}

func (tcp *TcpService) Start() {
	tcp.statusLock.Lock()
	if tcp.status & serviceEnable <= 0 {
		tcp.statusLock.Unlock()
		return
	}
	tcp.statusLock.Unlock()
	go func() {
		listen, err := net.Listen("tcp", tcp.Address)
		if err != nil {
			log.Errorf("tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		log.Infof("agent service start with: %s", tcp.Address)
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
	tcp.groups.close()
	tcp.agents.close()
	log.Debugf("tcp service closed.")
}

func (tcp *TcpService) SendPos(data []byte) {
	packData := pack(CMD_POS, data)
	tcp.agents.asyncSend(packData)
}

func (tcp *TcpService) keepalive() {
	for {
		select {
		case <-tcp.ctx.Ctx.Done():
			return
		default:
		}
		tcp.agents.asyncSend(packDataTickOk)
		tcp.groups.asyncSend(packDataTickOk)
		time.Sleep(time.Second * 3)
	}
}