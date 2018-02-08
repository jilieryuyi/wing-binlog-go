package main

import (
	"sync"
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
)
const tcpDefaultReadBufferSize = 4096
const CMD_SET_PRO = 1
const CMD_AUTH    = 2
const CMD_OK      = 3
const CMD_ERROR   = 4
const CMD_TICK    = 5
const CMD_EVENT   = 6

type Client struct {
	node *Node
	serviceIp string
	servicePort int
	isClose bool
	lock *sync.Mutex
	buffer []byte
	groupName string
}

type Node struct {
	conn *net.TCPConn
	isConnect bool
}

func NewClient(groupName string) *Client{
	agent := &Client{
		isClose  : true,
		node     : nil,
		lock     : new(sync.Mutex),
		buffer   : make([]byte, 0),
		groupName : groupName,
	}
	return agent
}

func (ag *Client) init() {
	ag.lock.Lock()
	defer ag.lock.Unlock()
	if ag.node != nil && ag.node.isConnect {
		ag.Close()
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ag.serviceIp, ag.servicePort))
	if err != nil {
		log.Panicf("start agent with error: %+v", err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	ag.node = &Node{
		conn:conn,
		isConnect:true,
	}
	if err != nil {
		log.Errorf("start agent with error: %+v", err)
		ag.node.isConnect = false
		ag.node.conn = nil
	} else {
		ag.isClose = false
	}
}

func (ag *Client) pack(cmd int, content string) []byte {
	// 数据打包
	c := []byte(content)
	l := len(c) + 2
	r := make([]byte, l + 4)

	// 4字节数据包长度
	r[0] = byte(l)
	r[1] = byte(l >> 8)
	r[2] = byte(l >> 16)
	r[3] = byte(l >> 32)

	// 2字节cmd
	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)

	// 实际美容
	r = append(r[:6], c...)
	return r
}

func (ag *Client) Start(serviceIp string, port int) {
	ag.serviceIp   = serviceIp
	ag.servicePort = port
	if ag.serviceIp == "" || ag.servicePort == 0 {
		log.Warnf("ip ang port empty")
		return
	}
	ag.init()
	keepalive := ag.pack(CMD_TICK, "")
	go func() {
		for {
			ag.node.conn.Write(keepalive)
			time.Sleep(time.Second * 5)
		}
	}()

	log.Debugf("====================client start====================")
	//握手包
	clientH := ag.pack(CMD_SET_PRO, ag.groupName)
	{
		var readBuffer [tcpDefaultReadBufferSize]byte
		for {
			ag.lock.Lock()
			if ag.isClose {
				ag.lock.Unlock()
				return
			}
			ag.lock.Unlock()
			if !ag.node.isConnect {
				ag.init()
			}
			if ag.node.conn == nil {
				time.Sleep(time.Second * 1)
				continue
			}
			//握手
			ag.node.conn.Write(clientH)
			for {
				ag.lock.Lock()
				if ag.isClose {
					ag.lock.Unlock()
					return
				}
				ag.lock.Unlock()
				buf := readBuffer[:tcpDefaultReadBufferSize]
				//清空旧数据 memset
				for i := range buf {
					buf[i] = byte(0)
				}
				size, err := ag.node.conn.Read(buf[0:])
				if err != nil || size <= 0 {
					log.Warnf("agent read with error: %+v", err)
					ag.disconnect()
					break
				}
				//log.Debugf("agent receive: %+v, %s", buf[:size], string(buf[:size]))
				ag.onMessage(buf[:size])
			}
		}
	}
}

func (ag *Client) disconnect() {
	ag.lock.Lock()
	defer ag.lock.Unlock()
	if ag.node == nil || !ag.node.isConnect {
		return
	}
	log.Warnf("---------------agent disconnect---------------")
	ag.node.conn.Close()
	ag.node.isConnect = false
}

func (ag *Client) Close() {
	if ag.isClose {
		log.Debugf("agent close was called, but not running")
		return
	}
	log.Warnf("---------------agent close---------------")
	ag.disconnect()
	ag.lock.Lock()
	ag.isClose = true
	ag.lock.Unlock()
}

func (ag *Client) onMessage(msg []byte) {
	ag.buffer = append(ag.buffer, msg...)
	//todo send broadcast
	//这里还需要解包数据
	for {
		bufferLen := len(ag.buffer)
		if bufferLen < 6 {
			return
		}
		//4字节长度，包含2自己的cmd
		contentLen := int(ag.buffer[0]) | int(ag.buffer[1]) << 8 | int(ag.buffer[2]) << 16 | int(ag.buffer[3]) << 24
		//2字节 command
		cmd := int(ag.buffer[4]) | int(ag.buffer[5]) << 8
		//数据未接收完整，等待下一次处理
		if bufferLen < 4 + contentLen {
			return
		}
		dataB := ag.buffer[6:4 + contentLen]
		log.Debugf("clen=%d, cmd=%d, %+v", contentLen, cmd, string(dataB))

		//switch(cmd) {
		//case CMD_EVENT:
		//	var data map[string] interface{}
		//	err := json.Unmarshal(dataB, &data)
		//	if err == nil {
		//		ag.tcp.SendAll(data)
		//	} else {
		//		log.Errorf("json Unmarshal error: %+v, %+v", dataB, err)
		//	}
		//default:
		//	ag.tcp.SendAll2(cmd, dataB)
		//}
		//数据移动，清除已读数据
		ag.buffer = append(ag.buffer[:0], ag.buffer[contentLen + 4:]...)
	}
}


func main() {

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.SetLevel(log.Level(5))

	groupName := "group1"
	serviceIp := "127.0.0.1"
	port := 10008

	client := NewClient(groupName)
	client.Start(serviceIp, port)
}