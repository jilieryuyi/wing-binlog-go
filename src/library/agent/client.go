package agent

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	"encoding/json"
	"library/app"
	"sync"
)

// 如果当前agent为leader
// agent client则断开
// 如果为leader，agent client则连接
// 连接后，server端的pos change事件就会通知到 client 端

type AgentClient struct {
	ctx *app.Context
	onPos []OnPosFunc
	buffer           []byte
	onEvent []OnEventFunc
	onRaw []OnRawFunc
	conn             *net.TCPConn
	statusLock       *sync.Mutex
	status int
	leader bool
	getLeader getLeaferFunc
}

type getLeaferFunc func()(string, int)
type OnEventFunc func(table string, data []byte) bool
type OnRawFunc func(msg []byte) bool
type AgentClientOption func(tcp *AgentClient)

func GetLeader(f getLeaferFunc) AgentClientOption{
	return func(tcp *AgentClient) {
		tcp.getLeader = f
	}
}

func newAgentClient(ctx *app.Context, opts ...AgentClientOption) *AgentClient {
	c := &AgentClient{
		ctx: ctx,
		buffer: make([]byte, 0),
		onEvent: make([]OnEventFunc, 0),
		conn: nil,
		statusLock:       new(sync.Mutex),
		status:           0,
		leader:false,
	}
	if len(opts) > 0 {
		for _, f := range opts {
			f(c)
		}
	}
	return c
}

func (tcp *AgentClient) keepalive() {
	data := pack(CMD_TICK, []byte(""))
	dl := len(data)
	for {
		select {
			case <-tcp.ctx.Ctx.Done():
				return
			default:
		}
		tcp.statusLock.Lock()
		if tcp.conn == nil ||
			tcp.status & agentStatusConnect <= 0 ||
			tcp.status & agentStatusOnline <= 0 {
				tcp.statusLock.Unlock()
			time.Sleep(3 * time.Second)
			continue
		}
		tcp.statusLock.Unlock()
		n, err := tcp.conn.Write(data)
		if err != nil {
			log.Errorf("agent keepalive error: %d, %v", n, err)
			tcp.disconnect()
		} else if n != dl {
			log.Errorf("%s send not complete", tcp.conn.RemoteAddr().String())
		}
		time.Sleep(3 * time.Second)
	}
}

func (tcp *AgentClient) OnLeader(leader bool) {
	log.Debugf("==============AgentClient OnLeader %v===============", leader)
	ip, port := tcp.getLeader()
	if ip == "" || port <= 0 {
		log.Errorf("ip or port empty: %v, %v", ip, port)
		return
	}
	if leader {
		// 断开client到 agent server的连接
		tcp.disconnect()
	} else {
		// 查询leader的 服务
		// 连接到agent server (leader)
		tcp.connect(ip, port)
	}
}

func (tcp *AgentClient) connect(ip string, port int) {
	if tcp.conn != nil {
		tcp.disconnect()
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		tcp.conn = nil
		return
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		tcp.conn = nil
		return
	}
	tcp.conn = conn
}

func (tcp *AgentClient) AgentStart(serviceIp string, port int) {
	agentH := PackPro(FlagAgent, []byte(""))
	hl := len(agentH)
	var readBuffer [tcpDefaultReadBufferSize]byte
	go func() {
		if serviceIp == "" || port == 0 {
			log.Warnf("ip or port empty %s:%d", serviceIp, port)
			return
		}

		tcp.statusLock.Lock()
		if tcp.status & agentStatusConnect > 0 {
			tcp.statusLock.Unlock()
			return
		}
		if tcp.status & agentStatusOnline <= 0 {
			tcp.status |= agentStatusOnline
		}
		tcp.statusLock.Unlock()

		for {
			select {
				case <-tcp.ctx.Ctx.Done():
					return
				default:
			}

			tcp.statusLock.Lock()
			if tcp.status & agentStatusOnline <= 0 {
				tcp.statusLock.Unlock()
				log.Warnf("agentStatusOffline return")
				return
			}
			tcp.statusLock.Unlock()

			tcp.connect(serviceIp, port)
			if tcp.conn == nil {
				time.Sleep(time.Second * 3)
				continue
			}

			tcp.statusLock.Lock()
			if tcp.status & agentStatusConnect <= 0 {
				tcp.status |= agentStatusConnect
			}
			tcp.statusLock.Unlock()

			log.Debugf("====================agent start %s:%d====================", serviceIp, port)
			// 简单的握手
			n, err := tcp.conn.Write(agentH)
			if err != nil {
				log.Warnf("write agent header data with error: %d, err", n, err)
				tcp.disconnect()
				continue
			}
			if n != hl {
				log.Errorf("%s tcp send not complete", tcp.conn.RemoteAddr().String())
			}
			for {
				log.Debugf("====agent is running====")

				tcp.statusLock.Lock()
				if tcp.status & agentStatusOnline <= 0 {
					tcp.statusLock.Unlock()
					log.Warnf("agentStatusOffline return - 2===%d:%d", tcp.status, tcp.status & agentStatusOnline)
					return
				}
				tcp.statusLock.Unlock()

				size, err := tcp.conn.Read(readBuffer[0:])
				//log.Debugf("read buffer len: %d, cap:%d", len(readBuffer), cap(readBuffer))
				if err != nil || size <= 0 {
					log.Warnf("agent read with error: %+v", err)
					tcp.disconnect()
					break
				}
				//log.Debugf("agent receive %d bytes: %+v, %s", size, readBuffer[:size], string(readBuffer[:size]))
				tcp.onMessage(readBuffer[:size])
				select {
				case <-tcp.ctx.Ctx.Done():
					return
				default:
				}
			}
		}
	}()
}

func (tcp *AgentClient) onMessage(msg []byte) {
	tcp.buffer = append(tcp.buffer, msg...)
	for {
		bufferLen := len(tcp.buffer)
		if bufferLen < 6 {
			return
		}
		//4字节长度，包含2自己的cmd
		contentLen := int(tcp.buffer[0]) | int(tcp.buffer[1]) << 8 | int(tcp.buffer[2]) << 16 | int(tcp.buffer[3]) << 24
		//2字节 command
		cmd := int(tcp.buffer[4]) | int(tcp.buffer[5]) << 8
		if !hasCmd(cmd) {
			log.Errorf("cmd %d dos not exists: %v, %s", cmd, tcp.buffer, string(tcp.buffer))
			tcp.buffer = make([]byte, 0)
			return
		}
		if bufferLen < 4 + contentLen {
			return
		}
		dataB := tcp.buffer[6:4 + contentLen]
		switch cmd {
		case CMD_EVENT:
			var data map[string] interface{}
			err := json.Unmarshal(dataB, &data)
			if err == nil {
				log.Debugf("agent receive event: %+v", data)
				//tcp.SendAll(data["table"].(string), dataB)
				for _, f := range tcp.onEvent {
					f(data["table"].(string), dataB)
				}
			} else {
				log.Errorf("json Unmarshal error: %+v, %s, %+v", dataB, string(dataB), err)
			}
		case CMD_TICK:
			//log.Debugf("keepalive: %s", string(dataB))
		case CMD_POS:
			log.Debugf("receive pos: %v", dataB)
			//for {
			//	if len(tcp.ctx.PosChan) < cap(tcp.ctx.PosChan) {
			//		break
			//	}
			//	log.Warnf("cache full, try wait")
			//}
			//tcp.ctx.PosChan <- string(dataB)
			if len(tcp.onPos) > 0 {
				for _, f := range tcp.onPos {
					f(dataB)
				}
			}
		default:
			//tcp.sendRaw(pack(cmd, msg))
			//log.Debugf("does not support")
			for _, f := range tcp.onRaw {
				f(pack(cmd, msg))
			}
		}
		if len(tcp.buffer) <= 0 {
			log.Errorf("tcp.buffer is empty")
			return
		}
		tcp.buffer = append(tcp.buffer[:0], tcp.buffer[contentLen+4:]...)
	}
}

func (tcp *AgentClient) disconnect() {
	tcp.statusLock.Lock()
	if tcp.conn == nil || tcp.status & agentStatusConnect <= 0 {
		tcp.statusLock.Unlock()
		log.Debugf("agent is in disconnect status")
		return
	}
	tcp.statusLock.Unlock()
	log.Warnf("====================agent disconnect====================")
	tcp.conn.Close()

	tcp.statusLock.Lock()
	if tcp.status & agentStatusConnect > 0 {
		tcp.status ^= agentStatusConnect
	}
	tcp.statusLock.Unlock()
}

func (tcp *AgentClient) AgentStop() {
	tcp.statusLock.Lock()
	if tcp.status & agentStatusOnline <= 0 {
		tcp.statusLock.Unlock()
		return
	}
	tcp.statusLock.Unlock()
	log.Warnf("====================agent close====================")
	tcp.disconnect()

	tcp.statusLock.Lock()
	if tcp.status & agentStatusOnline > 0 {
		tcp.status ^= agentStatusOnline
	}
	tcp.statusLock.Unlock()
}



