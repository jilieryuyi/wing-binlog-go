package control

import (
	"time"
	"net"
	log "github.com/sirupsen/logrus"
	"library/app"
	"sync"
)

func newNode(ctx *app.Context, conn *net.Conn) *TcpClientNode {
	node := &TcpClientNode{
		sendQueue:        make(chan []byte, 32),
		conn:             conn,
		connectTime:      time.Now().Unix(),
		recvBuf:          make([]byte, 0),
		status:           tcpNodeOnline | tcpNodeIsNormal,
		group:            "",
		ctx:              ctx,
		lock:             new(sync.Mutex),
	}
	return node
}

func (node *TcpClientNode) setGroup(group string) {
	node.group = group
}

func (node *TcpClientNode) changNodeType(nodeType int) {
	if node.status & nodeType > 0 {
		return
	}
	node.lock.Lock()
	defer node.lock.Unlock()
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

func (node *TcpClientNode) close() {
	node.lock.Lock()
	defer node.lock.Unlock()
	if node.status & tcpNodeOnline <= 0 {
		return
	}
	if node.status & tcpNodeOnline > 0{
		node.status ^= tcpNodeOnline
		(*node.conn).Close()
		close(node.sendQueue)
	}
}

func (node *TcpClientNode) send(data []byte) (int, error) {
	(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 3))
	return (*node.conn).Write(data)
}

func (node *TcpClientNode) asyncSend(data []byte) {
	node.lock.Lock()
	if node.status & tcpNodeOnline <= 0 {
		node.lock.Unlock()
		return
	}
	node.lock.Unlock()
	for {
		if len(node.sendQueue) < cap(node.sendQueue) {
			break
		}
		log.Warnf("cache full, try wait")
	}
	node.sendQueue <- data
}

func (node *TcpClientNode) setReadDeadline(t time.Time) {
	(*node.conn).SetReadDeadline(t)
}


