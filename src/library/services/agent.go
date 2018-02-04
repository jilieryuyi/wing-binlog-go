package services

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
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
	}

	//todo get service ip and port
	agent.serviceIp, agent.servicePort = tcp.Drive.GetLeader()

	agent.nodeInit()
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
	//connect pool
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	ag.node = &agentNode{
		conn:conn,
		isConnect:true,
	}
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		ag.node.isConnect = false
	}
}

func (ag *Agent) Start() {
	go func() {
		var readBuffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
		for {
			if !ag.node.isConnect {
				ag.nodeInit()
			}
			for {
				ag.lock.Lock()
				if ag.isClose {
					ag.lock.Unlock()
					return
				}
				ag.lock.Unlock()
				buf := readBuffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
				//清空旧数据 memset
				for i := range buf {
					buf[i] = byte(0)
				}
				size, err := ag.node.conn.Read(buf[0:])
				if err != nil {
					log.Warnf("agent error: %+v", err)
					ag.disconnect()
					break
				}
				log.Debugf("agent receive: %+v, %s", buf[:size], string(buf))
				ag.onMessage(buf[:size])
				select {
				case <-(*ag.tcp.ctx).Done():
					log.Warnf("agent context quit")
					return
				default:
				}
			}
		}
	}()
}

func (ag *Agent) disconnect() {
	if !ag.node.isConnect {
		return
	}
	//todo disconnect
	ag.node.conn.Close()
	ag.lock.Lock()
	ag.node.isConnect = false
	ag.lock.Unlock()
}

func (ag *Agent) Close() {
	ag.disconnect()
	ag.lock.Lock()
	ag.isClose = true
	ag.lock.Unlock()
}

func (ag *Agent) onMessage(msg []byte) {
	//todo send broadcast
	ag.tcp.SendAll(msg)
}
