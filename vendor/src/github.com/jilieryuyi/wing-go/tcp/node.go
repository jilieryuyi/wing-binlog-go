package tcp

import (
	"net"
	log "github.com/sirupsen/logrus"
	"sync"
	"io"
	"context"
)

const (
	tcpMaxSendQueue = 10000
	tcpNodeOnline   = 1 << iota
)
type NodeFunc   func(n *ClientNode)
type NodeOption func(n *ClientNode)

type ClientNode struct {
	conn              *net.Conn
	sendQueue         chan []byte
	recvBuf           []byte
	status            int
	wg                *sync.WaitGroup
	lock              *sync.Mutex
	onclose           []NodeFunc
	ctx               context.Context
	onMessageCallback []OnServerMessageFunc
	codec             ICodec
}

func setOnMessage(f ...OnServerMessageFunc) NodeOption {
	return func(n *ClientNode) {
		n.onMessageCallback = append(n.onMessageCallback, f...)
	}
}

func newNode(ctx context.Context, conn *net.Conn, codec ICodec, opts ...NodeOption) *ClientNode {
	node := &ClientNode{
		conn:              conn,
		sendQueue:         make(chan []byte, tcpMaxSendQueue),
		recvBuf:           make([]byte, 0),
		status:            tcpNodeOnline,
		ctx:               ctx,
		lock:              new(sync.Mutex),
		onclose:           make([]NodeFunc, 0),
		wg:                new(sync.WaitGroup),
		onMessageCallback: make([]OnServerMessageFunc, 0),
		codec:             codec,
	}
	for _, f := range opts {
		f(node)
	}
	go node.asyncSendService()
	return node
}

func setOnNodeClose(f NodeFunc) NodeOption {
	return func(n *ClientNode) {
		n.onclose = append(n.onclose, f)
	}
}

func (node *ClientNode) close() {
	node.lock.Lock()
	if node.status & tcpNodeOnline <= 0 {
		node.lock.Unlock()
		return
	}
	if node.status & tcpNodeOnline > 0{
		node.status ^= tcpNodeOnline
		(*node.conn).Close()
		close(node.sendQueue)
	}
	log.Warnf("node close")
	node.lock.Unlock()
	for _, f := range node.onclose {
		f(node)
	}
}

func (node *ClientNode) Send(msgId int64, data []byte) (int, error) {
	sendData := node.codec.Encode(msgId, data)
	return (*node.conn).Write(sendData)
}

func (node *ClientNode) AsyncSend(msgId int64, data []byte) {
	if node.status & tcpNodeOnline <= 0 {
		return
	}
	senddata := node.codec.Encode(msgId, data)
	node.sendQueue <- senddata
}

func (node *ClientNode) asyncSendService() {
	node.wg.Add(1)
	defer node.wg.Done()
	for {
		if node.status & tcpNodeOnline <= 0 {
			log.Info("tcp node is closed, clientSendService exit.")
			return
		}
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Info("tcp node sendQueue is closed, sendQueue channel closed.")
				return
			}
			size, err := (*node.conn).Write(msg)
			if err != nil {
				log.Errorf("tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
				node.close()
				return
			}
			if size != len(msg) {
				log.Errorf("%s send not complete: %v", (*node.conn).RemoteAddr().String(), msg)
			}
		case <- node.ctx.Done():
			log.Debugf("context is closed, wait for exit, left: %d", len(node.sendQueue))
			if len(node.sendQueue) <= 0 {
				log.Info("tcp service, clientSendService exit.")
				return
			}
		}
	}
}

func (node *ClientNode) onMessage(msg []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Unpack recover##########%+v, %+v", err, node.recvBuf)
			node.recvBuf = make([]byte, 0)
		}
	}()
	node.recvBuf = append(node.recvBuf, msg...)
	for {
		bufferLen := len(node.recvBuf)
		msgId, content, pos, err := node.codec.Decode(node.recvBuf)
		if err != nil {
			node.recvBuf = make([]byte, 0)
			log.Errorf("node.recvBuf error %v", err)
			return
		}
		if msgId <= 0 {
			return
		}
		if len(node.recvBuf) >= pos {
			node.recvBuf = append(node.recvBuf[:0], node.recvBuf[pos:]...)
		} else {
			node.recvBuf = make([]byte, 0)
			log.Errorf("pos %v(olen=%v) error, cmd=%v, content=%v(%v) len is %v, data is: %+v", pos, bufferLen, msgId, content, string(content), len(node.recvBuf), node.recvBuf)
		}
		for _, f := range node.onMessageCallback {
			f(node, msgId, content)
		}
	}
}

func (node *ClientNode) readMessage() {
	for {
		readBuffer := make([]byte, 4096)
		size, err := (*node.conn).Read(readBuffer)
		if err != nil && err != io.EOF {
			log.Warnf("tcp node disconnect with error: %v, %v", (*node.conn).RemoteAddr().String(), err)
			node.close()
			return
		}
		node.onMessage(readBuffer[:size])
	}
}


