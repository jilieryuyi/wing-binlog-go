package services

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
	"encoding/json"
)

//如果当前客户端为follower
//agent会使用客户端连接到leader
//leader发生事件的时候通过agent转发到连接follower的tcp客户端
//实现数据代理
//todo 这里应该弄一个连接池

type Agent struct {
	tcp *TcpService
	node *agentNode
	serviceIp string
	servicePort int
	isClose bool
	lock *sync.Mutex
	buffer []byte
}

type agentNode struct {
	conn *net.TCPConn
	isConnect bool
}

func newAgent(tcp *TcpService) *Agent{
	agent := &Agent{
		tcp:tcp,
		isClose:false,
		node:nil,
		lock:new(sync.Mutex),
		buffer:make([]byte, 0),
	}
	return agent
}

func (ag *Agent) nodeInit() {
	ag.lock.Lock()
	defer ag.lock.Unlock()
	if ag.node != nil && ag.node.isConnect {
		ag.Close()
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ag.serviceIp, ag.servicePort))
	if err != nil {
		log.Panicf("start agent with error: %+v", err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	ag.node = &agentNode{
		conn:conn,
		isConnect:true,
	}
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		ag.node.isConnect = false
		ag.node.conn = nil
	}
}

func (ag *Agent) Start(serviceIp string, port int) {
	//todo get service ip and port
	ag.serviceIp = serviceIp
	ag.servicePort = port//ag.tcp.GetLeader()
	if ag.serviceIp == "" || ag.servicePort == 0 {
		log.Warnf("ip ang port empty")
		return
	}
	ag.nodeInit()
	log.Debugf("====================agent start====================")
	agentH := ag.tcp.pack(CMD_AGENT, "")
	go func() {
		var readBuffer [tcpDefaultReadBufferSize]byte
		for {
			ag.lock.Lock()
			if ag.isClose {
				ag.lock.Unlock()
				return
			}
			ag.lock.Unlock()
			if !ag.node.isConnect {
				ag.nodeInit()
			}
			if ag.node.conn == nil {
				time.Sleep(time.Second * 1)
				continue
			}
			//握手
			ag.node.conn.Write(agentH)
			for {
				ag.lock.Lock()
				if ag.isClose {
					ag.lock.Unlock()
					return
				}
				ag.lock.Unlock()
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
				case <-ag.tcp.ctx.Ctx.Done():
					log.Warnf("agent context quit")
					return
				default:
				}
			}
		}
	}()
}

func (ag *Agent) disconnect() {
	ag.lock.Lock()
	defer ag.lock.Unlock()
	if ag.node == nil || !ag.node.isConnect {
		return
	}
	log.Warnf("---------------agent disconnect---------------")
	ag.node.conn.Close()
	ag.node.isConnect = false
}

func (ag *Agent) Close() {
	if ag.isClose {
		log.Debugf("agent close was called, but not running")
		return
	}
	log.Warnf("---------------agent close---------------")
	ag.disconnect()
	ag.lock.Lock()
	ag.isClose = true
	ag.lock.Unlock()
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

		switch(cmd) {
		case CMD_EVENT:
			var data map[string] interface{}
			err := json.Unmarshal(dataB, &data)
			if err == nil {
				ag.tcp.SendAll(data)
			} else {
				log.Errorf("json Unmarshal error: %+v, %+v", dataB, err)
			}
		default:
			ag.tcp.SendAll2(cmd, dataB)
		}
		//数据移动，清除已读数据
		ag.buffer = append(ag.buffer[:0], ag.buffer[contentLen + 4:]...)
	}
}
