package services

import (
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	"encoding/json"
)

func (tcp *TcpService) agentKeepalive() {
	data := pack(CMD_TICK, []byte(""))
	dl := len(data)
	for {
		select {
			case <-tcp.ctx.Ctx.Done():
				return
			default:
		}
		if tcp.conn == nil ||
			tcp.status & agentStatusDisconnect > 0 ||
			tcp.status & agentStatusOffline > 0 {
			time.Sleep(3 * time.Second)
			continue
		}
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

func (tcp *TcpService) connect(ip string, port int) {
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

func (tcp *TcpService) AgentStart(serviceIp string, port int) {
	agentH := PackPro(FlagAgent, []byte(""))
	hl := len(agentH)
	var readBuffer [tcpDefaultReadBufferSize]byte
	go func() {
		if serviceIp == "" || port == 0 {
			log.Warnf("ip or port empty %s:%d", serviceIp, port)
			return
		}

		tcp.lock.Lock()
		if tcp.status & agentStatusConnect > 0 {
			tcp.lock.Unlock()
			return
		}
		if tcp.status & agentStatusOffline > 0 {
			tcp.status ^= agentStatusOffline
			tcp.status |= agentStatusOnline
		}
		tcp.lock.Unlock()

		for {
			select {
				case <-tcp.ctx.Ctx.Done():
					return
				default:
			}

			tcp.lock.Lock()
			if tcp.status & agentStatusOffline > 0 {
				tcp.lock.Unlock()
				log.Warnf("agentStatusOffline return")
				return
			}
			tcp.lock.Unlock()

			tcp.connect(serviceIp, port)
			if tcp.conn == nil {
				log.Warnf("conn is nil")
				time.Sleep(time.Second * 3)
				continue
			}

			tcp.lock.Lock()
			if tcp.status & agentStatusDisconnect > 0 {
				tcp.status ^= agentStatusDisconnect
				tcp.status |= agentStatusConnect
			}
			tcp.lock.Unlock()

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

				tcp.lock.Lock()
				if tcp.status & agentStatusOffline > 0 {
					tcp.lock.Unlock()
					log.Warnf("agentStatusOffline return - 2===%d:%d", tcp.status, tcp.status&agentStatusOffline)
					return
				}
				tcp.lock.Unlock()

				size, err := tcp.conn.Read(readBuffer[0:])
				//log.Debugf("read buffer len: %d, cap:%d", len(readBuffer), cap(readBuffer))
				if err != nil || size <= 0 {
					log.Warnf("agent read with error: %+v", err)
					tcp.disconnect()
					break
				}
				//log.Debugf("agent receive %d bytes: %+v, %s", size, readBuffer[:size], string(readBuffer[:size]))
				tcp.onAgentMessage(readBuffer[:size])
				select {
				case <-tcp.ctx.Ctx.Done():
					return
				default:
				}
			}
		}
	}()
}

func (tcp *TcpService) onAgentMessage(msg []byte) {
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
				tcp.SendAll(data["table"].(string), dataB)
			} else {
				log.Errorf("json Unmarshal error: %+v, %s, %+v", dataB, string(dataB), err)
			}
		case CMD_TICK:
			//log.Debugf("keepalive: %s", string(dataB))
		case CMD_POS:
			log.Debugf("receive pos: %v", dataB)
			for {
				if len(tcp.ctx.PosChan) < cap(tcp.ctx.PosChan) {
					break
				}
				log.Warnf("cache full, try wait")
			}
			tcp.ctx.PosChan <- string(dataB)
		default:
			tcp.sendRaw(pack(cmd, msg))
		}
		if len(tcp.buffer) <= 0 {
			log.Errorf("tcp.buffer is empty")
			return
		}
		tcp.buffer = append(tcp.buffer[:0], tcp.buffer[contentLen+4:]...)
	}
}

func (tcp *TcpService) disconnect() {
	if tcp.conn == nil || tcp.status & agentStatusDisconnect > 0 {
		log.Debugf("agent is in disconnect status")
		return
	}
	log.Warnf("====================agent disconnect====================")
	tcp.conn.Close()

	tcp.lock.Lock()
	if tcp.status & agentStatusConnect > 0 {
		tcp.status ^= agentStatusConnect
		tcp.status |= agentStatusDisconnect
	}
	tcp.lock.Unlock()
}

func (tcp *TcpService) AgentStop() {
	if tcp.status & agentStatusOffline > 0 {
		return
	}
	log.Warnf("====================agent close====================")
	tcp.disconnect()

	tcp.lock.Lock()
	if tcp.status & agentStatusOnline > 0 {
		tcp.status ^= agentStatusOnline
		tcp.status |= agentStatusOffline
	}
	tcp.lock.Unlock()
}

