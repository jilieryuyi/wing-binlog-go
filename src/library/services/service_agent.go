package services

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
	"encoding/json"
	"library/app"
)

//如果当前客户端为follower
//agent会使用客户端连接到leader
//leader发生事件的时候通过agent转发到连接follower的tcp客户端
//实现数据代理

const (
	AgentStatusOffline = 1 << iota
	AgentStatusOnline
	AgentStatusConnect
	AgentStatusDisconnect
)

type Agent struct {
	node *agentNode
	serviceIp string
	servicePort int
	lock *sync.Mutex
	buffer []byte
	ctx    *app.Context
	sendAllChan1 chan map[string] interface{}
	sendAllChan2 chan []byte
	status int
}

type agentNode struct {
	conn *net.TCPConn
	//isConnect bool
}

func newAgent(ctx *app.Context, sendAllChan1 chan map[string] interface{}, sendAllChan2 chan []byte) *Agent{
	agent := &Agent{
		sendAllChan1 : sendAllChan1,
		sendAllChan2 : sendAllChan2,
		node    : nil,
		lock    : new(sync.Mutex),
		buffer  : make([]byte, 0),
		ctx     : ctx,
		//serviceChan: make(chan struct{}, 100),
		status  : AgentStatusOffline | AgentStatusDisconnect,
	}
	//go agent.lookAgent()
	return agent
}

func (ag *Agent) nodeInit() {
	if ag.node != nil && ag.node.conn != nil {
		ag.disconnect()
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ag.serviceIp, ag.servicePort))
	if err != nil {
		log.Panicf("start agent with error: %+v", err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	ag.node = &agentNode{
		conn:conn,
	}
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		ag.node.conn = nil
	}
}

func (ag *Agent) Start(serviceIp string, port int) {
	ag.status ^= AgentStatusOffline
	ag.status |= AgentStatusOnline

	if ag.status & AgentStatusConnect > 0 {
		log.Debugf("agent is still is running")
		return
	}

	ag.serviceIp   = serviceIp
	ag.servicePort = port
	if ag.serviceIp == "" || ag.servicePort == 0 {
		log.Warnf("ip and port empty")
		return
	}
	agentH := pack(CMD_AGENT, "")
	var readBuffer [tcpDefaultReadBufferSize]byte
	for {
		if ag.status & AgentStatusOffline > 0 {
			log.Debugf("AgentStatusOffline return")
			return
		}
		ag.nodeInit()
		if ag.node == nil || ag.node.conn == nil {
			log.Warnf("node | conn is nil")
			time.Sleep(time.Second * 3)
			continue
		}

		ag.status ^= AgentStatusDisconnect
		ag.status |= AgentStatusConnect

		log.Debugf("====================agent start====================")
		//握手
		ag.node.conn.Write(agentH)
		for {
			if ag.status & AgentStatusOffline > 0 {
				return
			}
			buf := readBuffer[:tcpDefaultReadBufferSize]
			//清空旧数据 memset
			for i := range buf {
				buf[i] = byte(0)
			}
			size, err := ag.node.conn.Read(buf[0:])
			if err != nil || size <= 0 {
				log.Warnf("agent read with error: %+v", err)
				ag.disconnect()
				break
			}
			log.Debugf("agent receive: %+v, %s", buf[:size], string(buf[:size]))
			ag.onMessage(buf[:size])
			select {
			case <-ag.ctx.Ctx.Done():
				log.Warnf("agent context quit")
			return
			default:
			}
		}
	}
}

func (ag *Agent) disconnect() {
	if ag.node == nil || ag.status & AgentStatusDisconnect > 0 {
		return
	}
	log.Warnf("---------------agent disconnect---------------")
	ag.node.conn.Close()
	ag.status ^= AgentStatusConnect
	ag.status |= AgentStatusDisconnect
}

func (ag *Agent) Close() {
	if ag.status & AgentStatusOffline > 0 {
		log.Debugf("agent close was called, but not running")
		return
	}
	log.Warnf("---------------agent close---------------")
	ag.disconnect()
	ag.status ^= AgentStatusOnline
	ag.status |= AgentStatusOffline
}

func (ag *Agent) onMessage(msg []byte) {
	ag.buffer = append(ag.buffer, msg...)
	//todo send broadcast
	//这里还需要解包数据
	for {
		bufferLen := len(ag.buffer)
		if bufferLen < 6 {
			return
		}
		//4字节长度，包含2自己的cmd
		contentLen := int(ag.buffer[0]) | int(ag.buffer[1]) << 8 | int(ag.buffer[2]) << 16 | int(ag.buffer[3]) << 24
		//2字节 command
		cmd := int(ag.buffer[4]) | int(ag.buffer[5]) << 8
		//数据未接收完整，等待下一次处理
		if bufferLen < 4 + contentLen {
			return
		}
		dataB := ag.buffer[6:4 + contentLen]
		log.Debugf("clen=%d, cmd=%d, %+v", contentLen, cmd, dataB)

		switch cmd {
		case CMD_EVENT:
			var data map[string] interface{}
			err := json.Unmarshal(dataB, &data)
			if err == nil {
				if len(ag.sendAllChan1) < cap(ag.sendAllChan1) {
					ag.sendAllChan1 <- data
				} else {
					log.Warnf("ag.sendAllChan1 was full")
				}
			} else {
				log.Errorf("json Unmarshal error: %+v, %+v", dataB, err)
			}
		default:
			if len(ag.sendAllChan2) < cap(ag.sendAllChan2) {
				ag.sendAllChan2 <- pack(cmd, string(msg))
			} else {
				log.Warnf("ag.sendAllChan2 was full")
			}
		}
		//数据移动，清除已读数据
		ag.buffer = append(ag.buffer[:0], ag.buffer[contentLen + 4:]...)
	}
}
