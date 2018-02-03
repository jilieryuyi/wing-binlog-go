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
	conn *net.TCPConn
	isConnect bool
}
// return leader tcp service ip and port, like "127.0.0.1:9989"
//func (ag *Agent) getLeader() string {
//	return ""
//}

func (ag *Agent) Start() {
	if ag.isConnect {
		ag.Close()
	}
	ag.isConnect = false
	//todo get leader service ip and port, connect to it
	leaderIp := ""
	leaderPort := 0
	if leaderIp == "" || leaderPort == 0 {
		return
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", leaderIp, leaderPort))
	if err != nil {
		log.Panicf("start agent with error: %+v", err)
	}
	ag.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Panicf("start agent with error: %+v", err)
	}
	ag.isConnect = true
	var readBuffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
	for {
		buf := readBuffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据 memset
		for i := range buf {
			buf[i] = byte(0)
		}
		size, err := ag.conn.Read(buf[0:])
		if err != nil {
			log.Warnf("agent error: %+v", err)
			ag.Close()
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

func (ag *Agent) Close() {
	if !ag.isConnect {
		return
	}
	//todo disconnect
	ag.conn.Close()
	ag.isConnect = false
}

func (ag *Agent) onMessage(msg []byte) {
	//todo send broadcast
	ag.tcp.SendAll(msg)
}
