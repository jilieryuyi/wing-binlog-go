package main

import (
	"sync"
	"net"
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	"encoding/json"
	"os"
	"strconv"
)

const (
	CMD_SET_PRO = iota // 注册客户端操作，加入到指定分组
	CMD_AUTH           // 认证（暂未使用）
	CMD_ERROR          // 错误响应
	CMD_TICK           // 心跳包
	CMD_EVENT          // 事件
	CMD_AGENT
	CMD_STOP
	CMD_RELOAD
	CMD_SHOW_MEMBERS
	CMD_POS
)

const (
	clientOffline = 1 << iota
	clientOnline
)

const (
	nodeOffline = 1 << iota
	nodeOnline
)

const (
	tcpDefaultReadBufferSize = 4096
)

func hasCmd(cmd int) bool {
	return cmd == CMD_SET_PRO ||
		cmd == CMD_AUTH ||
		cmd == CMD_ERROR||
		cmd == CMD_TICK ||
		cmd == CMD_EVENT||
		cmd == CMD_AGENT||
		cmd == CMD_STOP||
		cmd == CMD_RELOAD||
		cmd == CMD_SHOW_MEMBERS||
		cmd == CMD_POS
}

type Client struct {
	node *Node
	lock *sync.Mutex
	buffer []byte
	startTime int64
	Services []*service
	times int64
	status int
}

type Node struct {
	conn *net.TCPConn
	status int
}

func NewClient(s []*service) *Client{
	client := &Client{
		status    : clientOffline,
		node      : nil,
		lock      : new(sync.Mutex),
		buffer    : make([]byte, 0),
		startTime : time.Now().Unix(),
		Services  : s,
		times     : 0,
	}
	return client
}

func (client *Client) connect(ip string, port int) {
	log.Debugf("connect to %s:%d", ip, port)
	client.lock.Lock()
	defer client.lock.Unlock()
	if client.node != nil && client.node.status & nodeOnline > 0 {
		client.disconnect()
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Errorf("connect to %s:%d with error: %+v", ip, port, err)
		return
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	client.node = &Node{
		conn : conn,
		status : nodeOnline,
	}
	if err != nil {
		log.Errorf("start client with error: %+v", err)
		client.node.status ^= nodeOnline
		client.node.status |= nodeOffline
		client.node.conn = nil
	} else {
		if client.status & clientOffline > 0 {
			client.status ^= clientOffline
			client.status |= clientOnline
		}
	}
}

func (client *Client) pack(cmd int, content string) []byte {
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
	// 实际数据内容
	r = append(r[:6], c...)
	return r
}

func (client *Client) setPro(content string) []byte {
	// 数据打包
	c := []byte(content)
	l := len(c) + 3
	r := make([]byte, l + 4)
	// 4字节数据包长度
	r[0] = byte(l)
	r[1] = byte(l >> 8)
	r[2] = byte(l >> 16)
	r[3] = byte(l >> 32)
	// 2字节cmd
	r[4] = byte(CMD_SET_PRO)
	r[5] = byte(CMD_SET_PRO >> 8)
	r[6] = byte(0)
	// 实际数据内容
	r = append(r[:7], c...)
	return r
}

func (client *Client) keepalive() {
	data := client.pack(CMD_TICK, "")
	dl := len(data)
	go func() {
		for {
			if client.node == nil {
				time.Sleep(time.Second * 5)
				continue
			}
			client.lock.Lock()
			if client.node.conn != nil && client.node.status & nodeOnline > 0 {
				n, err := client.node.conn.Write(data)
				if err != nil {
					client.disconnect()
				} else if n != dl {
					log.Errorf("发送数据不完整")
				}
			}
			client.lock.Unlock()
			time.Sleep(time.Second * 5)
		}
	}()
}

func (client *Client) Start() {
	client.keepalive()
	var readBuffer [tcpDefaultReadBufferSize]byte
	for {
		for _, server := range client.Services {
			client.connect(server.ip, server.port)
			if  client.node == nil || client.node.conn == nil || client.node.status & nodeOffline > 0 {
				time.Sleep(time.Second)
				continue
			}
			log.Debugf("====================client start====================")
			if client.status & clientOffline > 0 {
				return
			}
			//加入到分组
			clientH := client.setPro(server.groupName)
			client.node.conn.Write(clientH)
			for {
				if client.status & clientOffline > 0 {
					return
				}
				size, err := client.node.conn.Read(readBuffer[0:])
				if err != nil || size <= 0 {
					log.Warnf("client read with error: %+v", err)
					client.disconnect()
					break
				}
				client.onMessage(readBuffer[:size])
			}
		}
	}
}

func (client *Client) disconnect() {
	client.lock.Lock()
	defer client.lock.Unlock()
	if client.node == nil || client.node.status & nodeOffline > 0 {
		return
	}
	log.Warnf("---------------client disconnect---------------")
	client.node.conn.Close()
	if client.node.status & nodeOnline > 0 {
		client.node.status ^= nodeOnline
		client.node.status |= nodeOffline
	}
}

func (client *Client) Close() {
	if client.status & clientOffline > 0 {
		log.Debugf("client close was called, but not running")
		return
	}
	log.Warnf("---------------client close---------------")
	client.disconnect()
	client.lock.Lock()
	if client.status & clientOnline > 0 {
		client.status ^= clientOnline
		client.status |= clientOffline
	}
	client.lock.Unlock()
}

func (client *Client) onMessage(msg []byte) {
	client.buffer = append(client.buffer, msg...)
	for {
		bufferLen := len(client.buffer)
		if bufferLen < 6 {
			return
		}
		//4字节长度，包含2自己的cmd
		contentLen := int(client.buffer[0]) | int(client.buffer[1]) << 8 | int(client.buffer[2]) << 16 | int(client.buffer[3]) << 24
		//2字节 command
		cmd := int(client.buffer[4]) | int(client.buffer[5]) << 8
		if !hasCmd(cmd) {
			log.Errorf("cmd=%d 不支持的cmd事件", cmd)
			client.buffer = make([]byte, 0)
			return
		}
		//数据未接收完整，等待下一次处理
		if bufferLen < 4 + contentLen {
			return
		}
		dataB := client.buffer[6:4 + contentLen]
		switch cmd {
		case CMD_EVENT:
			client.times++
			log.Debugf("收到%d次数据库事件", client.times)
			p := int64(0)
			sp := time.Now().Unix() - client.startTime
			if sp > 0 {
				p = int64(client.times/sp)
			}
			log.Debugf("每秒接收数据 %d 条", p)
			var data interface{}
			json.Unmarshal(dataB, &data)
			log.Debugf("%+v", data)
		case CMD_SET_PRO:
		case CMD_AUTH:          // 认证（暂未使用）
		case CMD_ERROR:         // 错误响应
		case CMD_TICK:        // 心跳包
		case CMD_AGENT:
		case CMD_STOP:
		case CMD_RELOAD:
		case CMD_SHOW_MEMBERS:
		case CMD_POS:
		default:
			log.Errorf("cmd=%d 不支持的cmd事件", cmd)
			client.buffer = make([]byte, 0)
			return
		}
		//清除已读数据
		client.buffer = append(client.buffer[:0], client.buffer[contentLen + 4:]...)
	}
}


type service struct {
	groupName string
	ip string
	port int
}

func main() {
	//初始化debug终端输出日志支持
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.SetLevel(log.Level(5))
	defaultIp := "127.0.0.1"
	defaultPort := 9998

	if len(os.Args) >= 3 {
		defaultIp = os.Args[1]
		port, _:= strconv.Atoi(os.Args[2])
		defaultPort = port
	}

	ser1 := &service{
		groupName : "group1",
		ip : defaultIp,
		port : defaultPort,
	}

	s := make([]*service, 0)
	s = append(s, ser1)

	client := NewClient(s)
	client.Start()
}