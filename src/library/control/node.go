package control

import (
	"time"
	"net"
	"library/app"
	"sync"
)

func newNode(ctx *app.Context, conn *net.Conn) *TcpClientNode {
	node := &TcpClientNode{
		conn:             conn,
		recvBuf:          make([]byte, 0),
		status:           tcpNodeOnline | tcpNodeIsControl,
		ctx:              ctx,
		lock:             new(sync.Mutex),
	}
	return node
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
	}
}

func (node *TcpClientNode) send(data []byte) (int, error) {
	if node.status & tcpNodeOnline <= 0 {
		return 0, nodeOffline
	}
	(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 3))
	return (*node.conn).Write(data)
}

func (node *TcpClientNode) setReadDeadline(t time.Time) {
	(*node.conn).SetReadDeadline(t)
}


