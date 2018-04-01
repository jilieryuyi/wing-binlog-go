package control

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
	"library/app"
	"io"
)

func NewControl(ctx *app.Context, opts ...ControlOption) *TcpService {
	tcp := &TcpService{
		Address:               ctx.AppConfig.ControlListen,
		lock:             new(sync.Mutex),
		wg:               new(sync.WaitGroup),
		listener:         nil,
		ctx:              ctx,
		token:            app.GetKey(app.CachePath + "/token"),
	}
	for _, f := range opts {
		f(tcp)
	}
	return tcp
}

func ShowMember(f ShowMemberFunc) ControlOption {
	return func(tcp *TcpService){
		tcp.showmember = f
	}
}

func Reload(f ReloadFunc) ControlOption {
	return func(tcp *TcpService){
		tcp.reload = f
	}
}

func Stop(f StopFunc) ControlOption {
	return func(tcp *TcpService){
		tcp.stop = f
	}
}

func (tcp *TcpService) onClose(node *TcpClientNode) {
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	node.close()
}

func (tcp *TcpService) onConnect(conn *net.Conn) {
	node := newNode(tcp.ctx, conn)
	node.setReadDeadline(time.Now().Add(time.Second * 3))
	var readBuffer [tcpDefaultReadBufferSize]byte
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	for {
		size, err := (*conn).Read(readBuffer[0:])
		if err != nil {
			if err != io.EOF {
				log.Warnf("tcp node %s disconnect with error: %v", (*conn).RemoteAddr().String(), err)
			} else {
				log.Debugf("tcp node %s disconnect with error: %v", (*conn).RemoteAddr().String(), err)
			}
			tcp.onClose(node)
			return
		}
		log.Debugf("receive msg: %v, %+v", size, readBuffer[:size])
		tcp.onMessage(node, readBuffer[:size])
	}
}

// receive a new message
func (tcp *TcpService) onMessage(node *TcpClientNode, msg []byte) {
	log.Debugf("receive msg: %+v", msg)
	node.recvBuf = append(node.recvBuf, msg...)
	for {
		size := len(node.recvBuf)
		if size < 6 {
			return
		}
		clen := int(node.recvBuf[0]) | int(node.recvBuf[1]) << 8 |
			int(node.recvBuf[2]) << 16 | int(node.recvBuf[3]) << 24
		log.Debugf("len: %+v", clen)
		if len(node.recvBuf) < 	clen + 4 {
			return
		}
		cmd  := int(node.recvBuf[4]) | int(node.recvBuf[5]) << 8
		log.Debugf("cmd: %v", cmd)
		if !hasCmd(cmd) {
			log.Errorf("cmd %d does not exists, data: %v", cmd, node.recvBuf)
			node.recvBuf = make([]byte, 0)
			return
		}
		content := node.recvBuf[6 : clen + 4]
		switch cmd {
		case CMD_TICK:
			node.send(packDataTickOk)
		case CMD_STOP:
			log.Debugf("receive stop cmd")
			tcp.stop()
			node.send(pack(CMD_STOP, []byte("ok")))
		case CMD_RELOAD:
			tcp.reload(string(content))
			node.send(pack(CMD_RELOAD, []byte("ok")))
		case CMD_SHOW_MEMBERS:
			members := tcp.showmember()
			node.send(pack(CMD_SHOW_MEMBERS, []byte(members)))
		default:
			node.send(pack(CMD_ERROR, []byte(fmt.Sprintf("tcp service does not support cmd: %d", cmd))))
			node.recvBuf = make([]byte, 0)
			return
		}
		node.recvBuf = append(node.recvBuf[:0], node.recvBuf[clen + 4:]...)
	}
}

func (tcp *TcpService) Start() {
	go func() {
		listen, err := net.Listen("tcp", tcp.Address)
		if err != nil {
			log.Errorf("tcp service listen with error: %+v", err)
			return
		}
		tcp.listener = &listen
		for {
			conn, err := listen.Accept()
			select {
			case <-tcp.ctx.Ctx.Done():
				return
			default:
			}
			if err != nil {
				log.Warnf("tcp service accept with error: %+v", err)
				continue
			}
			go tcp.onConnect(&conn)
		}
	}()
}

func (tcp *TcpService) Close() {
	log.Debugf("tcp service closing, waiting for buffer send complete.")
	tcp.lock.Lock()
	defer tcp.lock.Unlock()
	if tcp.listener != nil {
		(*tcp.listener).Close()
	}
	log.Debugf("tcp service closed.")
}

