package services

import (
	"time"
	"net"
)

func newNode(conn *net.Conn) *tcpClientNode {
	node := &tcpClientNode{
		conn:             conn,
		sendQueue:        make(chan []byte, tcpMaxSendQueue),
		sendFailureTimes: 0,
		connectTime:      time.Now().Unix(),
		sendTimes:        int64(0),
		recvBuf:          make([]byte, 0),
		status:           tcpNodeOnline | tcpNodeIsNormal,
		group:            "",
	}
	return node
}

func (node *tcpClientNode) setGroup(group string) {
	node.group = group
}

func (node *tcpClientNode) changNodeType(nodeType int) {
	if node.status & nodeType > 0 {
		return
	}
	if node.status & tcpNodeIsNormal > 0 {
		node.status ^= tcpNodeIsNormal
		node.status |= nodeType
		return
	}
	if node.status & tcpNodeIsAgent > 0 {
		node.status ^= tcpNodeIsAgent
		node.status |= nodeType
		return
	}
	if node.status & tcpNodeIsControl > 0 {
		node.status ^= tcpNodeIsControl
		node.status |= nodeType
		return
	}
}

func (node *tcpClientNode) close() {
	if node.status & tcpNodeOffline > 0 {
		return
	}
	if node.status & tcpNodeOnline > 0{
		node.status ^= tcpNodeOnline
		node.status |= tcpNodeOffline
		(*node.conn).Close()
	}
}

func (node *tcpClientNode) send(data []byte) (int, error) {
	(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 3))
	return (*node.conn).Write(data)
}
