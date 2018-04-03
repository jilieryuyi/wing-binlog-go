package tcp

import (
	"time"
	"net"
	log "github.com/sirupsen/logrus"
	"library/app"
	"sync"
	"io"
	"fmt"
	"sync/atomic"
)

func newNode(ctx *app.Context, conn *net.Conn, opts ...NodeOption) *tcpClientNode {
	node := &tcpClientNode{
		conn:             conn,
		sendQueue:        make(chan []byte, tcpMaxSendQueue),
		sendFailureTimes: 0,
		connectTime:      time.Now().Unix(),
		recvBuf:          make([]byte, 0),
		status:           tcpNodeOnline | tcpNodeIsNormal,
		group:            "",
		ctx:              ctx,
		lock:             new(sync.Mutex),
		onclose:          make([]NodeFunc, 0),
		wg:               new(sync.WaitGroup),
	}
	node.setReadDeadline(time.Now().Add(time.Second * 3))
	for _, f := range opts {
		f(node)
	}
	return node
}

func NodeClose(f NodeFunc) NodeOption {
	return func(n *tcpClientNode) {
		n.onclose = append(n.onclose, f)
	}
}

func NodePro(f SetProFunc) NodeOption {
	return func(n *tcpClientNode) {
		n.onpro = f
	}
}

func (node *tcpClientNode) setGroup(group string) {
	node.group = group
}

func (node *tcpClientNode) changNodeType(nodeType int) {
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

func (node *tcpClientNode) close() {
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

	for _, f := range node.onclose {
		f(node)
	}
}

func (node *tcpClientNode) send(data []byte) (int, error) {
	(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 3))
	return (*node.conn).Write(data)
}

func (node *tcpClientNode) asyncSend(data []byte) {
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
		log.Warnf("cache full, try wait, %v, %v", len(node.sendQueue) , cap(node.sendQueue))
	}
	node.sendQueue <- data
}

func (node *tcpClientNode) setReadDeadline(t time.Time) {
	(*node.conn).SetReadDeadline(t)
}

func (node *tcpClientNode) onConnect() {
	var readBuffer [tcpDefaultReadBufferSize]byte
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	for {
		size, err := (*node.conn).Read(readBuffer[0:])
		if err != nil {
			if err != io.EOF {
				log.Warnf("tcp node %s disconnect with error: %v", (*node.conn).RemoteAddr().String(), err)
			} else {
				log.Debugf("tcp node %s disconnect with error: %v", (*node.conn).RemoteAddr().String(), err)
			}
			node.close()//tcp.onClose(node)
			return
		}
		//log.Debugf("tcp receive: %v", readBuffer[:size])
		node.onMessage(readBuffer[:size])
	}
}

func (node *tcpClientNode) onMessage(msg []byte) {
	node.recvBuf = append(node.recvBuf, msg...)
	for {
		size := len(node.recvBuf)
		if size < 6 {
			return
		}
		clen := int(node.recvBuf[0]) | int(node.recvBuf[1]) << 8 |
			int(node.recvBuf[2]) << 16 | int(node.recvBuf[3]) << 24
		if len(node.recvBuf) < 	clen + 4 {
			return
		}
		cmd  := int(node.recvBuf[4]) | int(node.recvBuf[5]) << 8
		if !hasCmd(cmd) {
			log.Errorf("cmd %d does not exists, data: %v", cmd, node.recvBuf)
			node.recvBuf = make([]byte, 0)
			return
		}
		content := node.recvBuf[6 : clen + 4]
		switch cmd {
		case CMD_SET_PRO:
			node.onSetProEvent(content)
		case CMD_TICK:
			node.asyncSend(packDataTickOk)
			//case CMD_STOP:
			//	tcp.ctx.Stop()
			//case CMD_RELOAD:
			//	tcp.ctx.Reload(string(content))
			//case CMD_SHOW_MEMBERS:
			//	tcp.onShowMembersEvent(node)
		default:
			node.asyncSend(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service does not support cmd: %d", cmd))))
			node.recvBuf = make([]byte, 0)
			return
		}
		node.recvBuf = append(node.recvBuf[:0], node.recvBuf[clen + 4:]...)
	}
}

func (node *tcpClientNode) onSetProEvent(data []byte) {
	// 客户端角色分为三种
	// 一种是普通的客户端
	// 一种是用来控制进程的客户端
	// 还有另外一种就是agent代理客户端
	flag    := data[0]
	content := string(data[1:])
	switch flag {
	case FlagSetPro:
		node.onSetPro(content)
		//case FlagControl:
		//	tcp.onControl(node, content)
	//case FlagAgent:
	//	node.setReadDeadline(time.Time{})
	//	node.send(packDataSetPro)
	//	node.changNodeType(tcpNodeIsAgent)
	//	go node.asyncSendService()
	case FlagPing:
		log.Debugf("receive ping data")
		node.send(packDataSetPro)
		node.close()
	default:
		node.close()
	}
}


func (node *tcpClientNode) onSetPro(groupName string) {
	//for _, f := range node.onpro {
	//	f(node)
	//}
	if !node.onpro(node, groupName) {
		node.send(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service, group does not exists: %s", groupName))))
		node.close()
		return
	}
	//group, found := tcp.groups[groupName]
	//if !found || groupName == "" {
	//	node.send(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service, group does not exists: %s", groupName))))
	//	node.close()
	//	return
	//}
	node.setReadDeadline(time.Time{})
	node.send(packDataSetPro)
	node.setGroup(groupName)
	//group.append(node)
	go node.asyncSendService()
}

func (node *tcpClientNode) asyncSendService() {
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
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 30))
			size, err := (*node.conn).Write(msg)
			if err != nil {
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Errorf("tcp send to %s error: %v", (*node.conn).RemoteAddr().String(), err)
				//tcp.onClose(node)
				node.close()
				return
			}
			if size != len(msg) {
				log.Errorf("%s send not complete: %v", (*node.conn).RemoteAddr().String(), msg)
			}
		case <-node.ctx.Ctx.Done():
			log.Debugf("context is closed, wait for exit, left: %d", len(node.sendQueue))
			if len(node.sendQueue) <= 0 {
				log.Info("tcp service, clientSendService exit.")
				return
			}
		}
	}
}




