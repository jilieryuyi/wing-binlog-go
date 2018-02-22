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
	node         *agentNode
	lock         *sync.Mutex
	buffer       []byte
	ctx          *app.Context
	sendAllChan1 chan map[string] interface{}
	sendAllChan2 chan []byte
	status       int
	last         int64
}

type agentNode struct {
	conn *net.TCPConn
}

func newAgent(ctx *app.Context, sendAllChan1 chan map[string] interface{}, sendAllChan2 chan []byte) *Agent{
	agent := &Agent{
		sendAllChan1 : sendAllChan1,
		sendAllChan2 : sendAllChan2,
		node    : nil,
		lock    : new(sync.Mutex),
		buffer  : make([]byte, 0),
		ctx     : ctx,
		status  : AgentStatusOffline | AgentStatusDisconnect,
	}
	go agent.keepalive()
	return agent
}

func (ag *Agent) keepalive() {
	data := pack(CMD_TICK, "agent keep alive")
	for {
		if ag.node == nil || ag.node.conn == nil ||
			ag.status & AgentStatusDisconnect > 0 ||
			ag.status & AgentStatusOffline > 0 {
			time.Sleep(3 * time.Second)
			continue
		}
		log.Debugf("agent keepalive")
		n, err := ag.node.conn.Write(data)
		if n <= 0 || err != nil {
			log.Errorf("agent keepalive error: %d, %v", n, err)
			ag.disconnect()
		}
		time.Sleep(3 * time.Second)
	}
}

func (ag *Agent) nodeInit(ip string, port int) {
	if ag.node != nil && ag.node.conn != nil {
		ag.disconnect()
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ip, port))
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
	if serviceIp == "" || port == 0 {
		log.Warnf("ip or port empty %s:%d", serviceIp, port)
		return
	}
	if ag.status & AgentStatusConnect > 0 {
		//if time.Now().Unix() - ag.last > 60 {
		//	log.Warnf("agent is timeout")
		//	ag.Close()
		//}
		log.Debugf("agent is still is running")
		return
	}
	ag.lock.Lock()
	if ag.status & AgentStatusOffline > 0 {
		ag.status ^= AgentStatusOffline
		ag.status |= AgentStatusOnline
	}
	ag.lock.Unlock()

	agentH := pack(CMD_AGENT, "")
	var readBuffer [tcpDefaultReadBufferSize]byte
	for {
		select {
		case <- ag.ctx.Ctx.Done():
			return
		default:
		}
		if ag.status & AgentStatusOffline > 0 {
			log.Warnf("AgentStatusOffline return")
			return
		}
		ag.nodeInit(serviceIp, port)
		if ag.node == nil || ag.node.conn == nil {
			log.Warnf("node | conn is nil")
			time.Sleep(time.Second * 3)
			continue
		}
		ag.lock.Lock()
		if ag.status & AgentStatusDisconnect > 0 {
			ag.status ^= AgentStatusDisconnect
			ag.status |= AgentStatusConnect
		}
		ag.lock.Unlock()
		log.Debugf("====================agent start %s:%d====================", serviceIp, port)
		// 简单的握手
		n, err := ag.node.conn.Write(agentH)
		if n <= 0 || err != nil {
			log.Warnf("write agent header data with error: %d, err", n, err)
			ag.disconnect()
			continue
		}
		for {
			log.Debugf("====agent is running====")
			ag.last = time.Now().Unix()
			if ag.status & AgentStatusOffline > 0 {
				log.Warnf("AgentStatusOffline return - 2===%d:%d", ag.status, ag.status & AgentStatusOffline)
				return
			}
			buf := readBuffer[0:]
			//clear data
			for i := range buf {
				buf[i] = byte(0)
			}
			size, err := ag.node.conn.Read(buf[0:])
			if err != nil || size <= 0 {
				log.Warnf("agent read with error: %+v", err)
				ag.disconnect()
				break
			}
			log.Debugf("agent receive %d bytes: %+v, %s", size, buf[:size], string(buf[:size]))
			ag.onMessage(buf[:size])
			select {
				case <-ag.ctx.Ctx.Done():
					return
				default:
			}
		}
	}
}

func (ag *Agent) disconnect() {
	if ag.node == nil || ag.status & AgentStatusDisconnect > 0 {
		log.Debugf("agent is in disconnect status")
		return
	}
	log.Warnf("====================agent disconnect====================")
	ag.node.conn.Close()

	ag.lock.Lock()
	if ag.status & AgentStatusConnect > 0 {
		ag.status ^= AgentStatusConnect
		ag.status |= AgentStatusDisconnect
	}
	ag.lock.Unlock()
}

func (ag *Agent) Close() {
	ag.lock.Lock()
	if ag.status & AgentStatusOffline > 0 {
		ag.lock.Unlock()
		log.Debugf("agent close was called, but not running")
		return
	}
	ag.lock.Unlock()

	log.Warnf("====================agent close====================")
	ag.disconnect()

	ag.lock.Lock()
	if ag.status & AgentStatusOnline > 0 {
		ag.status ^= AgentStatusOnline
		ag.status |= AgentStatusOffline
	}
	ag.lock.Unlock()
}

func (ag *Agent) onMessage(msg []byte) {
	ag.buffer = append(ag.buffer, msg...)
	for {
		bufferLen := len(ag.buffer)
		if bufferLen < 6 {
			return
		}
		//4字节长度，包含2自己的cmd
		contentLen := int(ag.buffer[0]) | int(ag.buffer[1]) << 8 | int(ag.buffer[2]) << 16 | int(ag.buffer[3]) << 24
		//2字节 command
		cmd := int(ag.buffer[4]) | int(ag.buffer[5]) << 8
		log.Debugf("bufferLen=%d, contentLen=%d, cmd=%d", bufferLen, contentLen, cmd)
		//数据未接收完整，等待下一次处理
		if bufferLen < 4 + contentLen {
			return
		}
		log.Debugf("%v", ag.buffer)
		dataB := ag.buffer[6:4 + contentLen]
		log.Debugf("clen=%d, cmd=%d, (%d)%+v", contentLen, cmd, len(dataB), dataB)

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
			log.Debugf("%+v", data)
		case CMD_TICK:
			log.Debugf("keepalive: %s", string(dataB))
		default:
			if len(ag.sendAllChan2) < cap(ag.sendAllChan2) {
				ag.sendAllChan2 <- pack(cmd, string(msg))
			} else {
				log.Warnf("ag.sendAllChan2 was full")
			}
		}
		//remove(&ag.buffer, contentLen + 4)
		log.Debugf("%d, contentLen + 4=%d", len(ag.buffer), contentLen + 4)
		if len(ag.buffer) >= contentLen + 4 {
			ag.buffer = append(ag.buffer[:0], ag.buffer[contentLen+4:]...)
		} else {
			log.Warnf("content len error")
		}
		log.Debugf("%v", ag.buffer)
	}
}
