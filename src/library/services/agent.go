package services

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
)

//如果当前客户端为follower
//agent会使用客户端连接到leader
//leader发生事件的时候通过agent转发到连接follower的tcp客户端
//实现数据代理
//todo 这里应该弄一个连接池

type Agent struct {
	tcp *TcpService
	nodes []*agentNode
	serviceIp string
	servicePort int
}

type agentNode struct {
	conn *net.TCPConn
	isConnect bool
}

// return leader tcp service ip and port, like "127.0.0.1:9989"
//func (ag *Agent) getLeader() string {
//	return ""
//}

func newAgent(tcp *TcpService) *Agent{
	agent := &Agent{
		tcp:tcp,
		nodes:make([]*agentNode, 4),
	}

	//todo get service ip and port

	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", agent.serviceIp, agent.servicePort))
	if err != nil {
		log.Panicf("start agent with error: %+v", err)
	}
	//connect pool
	for i := 0; i < 4; i++ {
		conn, err := net.DialTCP("tcp", nil, tcpAddr)
		agent.nodes[i] = &agentNode{
			conn:conn,
			isConnect:true,
		}
		if err != nil {
			log.Errorf("start agent with error: %+v", err)
			agent.nodes[i].isConnect = false
		}
	}
	return agent
}

func (ag *Agent) getConn() *agentNode {
	for i := 0; i < 4; i++ {
		if ag.nodes[i].isConnect {
			return ag.nodes[i]//.conn
		}
	}
	return nil
}

func (ag *Agent) Start() {
	node := ag.getConn()
	var readBuffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
	for {
		buf := readBuffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据 memset
		for i := range buf {
			buf[i] = byte(0)
		}
		size, err := node.conn.Read(buf[0:])
		if err != nil {
			log.Warnf("agent error: %+v", err)
			ag.Close(node)
			return
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

func (ag *Agent) Close(node *agentNode) {
	if !node.isConnect {
		return
	}
	//todo disconnect
	node.conn.Close()
	node.isConnect = false
}

func (ag *Agent) onMessage(msg []byte) {
	//todo send broadcast
	ag.tcp.SendAll(msg)
}
