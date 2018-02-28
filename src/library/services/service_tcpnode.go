package services

import (
	"time"
	"net"
	"sync/atomic"
	"sync"
	log "github.com/sirupsen/logrus"
	"library/app"
)

func newNode(ctx *app.Context, conn *net.Conn) *tcpClientNode {
	node := &tcpClientNode{
		conn:             conn,
		sendQueue:        make(chan []byte, tcpMaxSendQueue),
		sendFailureTimes: 0,
		connectTime:      time.Now().Unix(),
		sendTimes:        int64(0),
		recvBuf:          make([]byte, 0),
		status:           tcpNodeOnline | tcpNodeIsNormal,
		group:            "",
		wg:               new(sync.WaitGroup),
		ctx:              ctx,
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
	node.wg.Wait()
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

func (node *tcpClientNode) asyncSend(data []byte) {
	//log.Debugf("send to node")
	if node.status & tcpNodeOffline > 0 {
		return
	}
	for {
		if len(node.sendQueue) < cap(node.sendQueue) {
			break
		}
		log.Warnf("cache full, try wait")
	}
	node.sendQueue <- data
}

func (node *tcpClientNode) asyncSendService() {
	node.wg.Add(1)
	defer node.wg.Done()
	for {
		if node.status & tcpNodeOffline > 0 {
			log.Info("tcp service, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("tcp service, sendQueue channel closed.")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 3))
			size, err := (*node.conn).Write(msg)
			atomic.AddInt64(&node.sendTimes, int64(1))
			if size <= 0 || err != nil {
				//atomic.AddInt64(&tcp.sendFailureTimes, int64(1))
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Errorf("tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
			}
		case <-node.ctx.Ctx.Done():
			if len(node.sendQueue) <= 0 {
				log.Info("tcp service, clientSendService exit.")
				return
			}
		}
	}
}

func (node *tcpClientNode) setReadDeadline(t time.Time) {
	(*node.conn).SetReadDeadline(t)
}


