package client

import (
	"sync"
	"net"
	log "github.com/sirupsen/logrus"
	"time"
	"encoding/json"
	"fmt"
	"strings"
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
	Services map[string]*serverNode//string
	times int64
	status int
	onevent []OnEventFunc
	topics []string
	consulAddress string
	getConnects func(ip string, port int) uint64
}

type serverNode struct {
	offline bool
	host string
	port int
	connects uint64
}

type Node struct {
	conn *net.TCPConn
	status int
}
type wait struct {
	c chan struct{}
	closed bool
}

type ClientOption func(client *Client)
type OnEventFunc func(data map[string]interface{})

func NewClient(opts ...ClientOption) *Client{
	client := &Client{
		status    : clientOffline,
		node      : nil,
		lock      : new(sync.Mutex),
		buffer    : make([]byte, 0),
		startTime : time.Now().Unix(),
		Services  : make(map[string]*serverNode),
		times     : 0,
		onevent   : make([]OnEventFunc, 0),
		topics    : make([]string,0),
		getConnects: func(ip string, port int) uint64 {
			return 0
		},
	}
	for _, f := range opts {
		f(client)
	}
	var wi = &wait{
		c:make(chan struct{}),
		closed:false,
	}
	if client.Services != nil && len(client.Services) > 0 {
		go client.start(wi)
	} else if client.consulAddress != "" {
		//获取所有的服务
		w := newWatch(client.consulAddress, onWatch(func(ip string, port int, event int) {
			client.lock.Lock()
			defer client.lock.Unlock()
			defer log.Debugf("2-current services list: %+v", client.Services)

			s := fmt.Sprintf("%v:%v", ip, port)
			switch event {
			case EV_CHANGE:
				log.Debugf("service change(delete): %s", s)
				delete(client.Services, s)
			case EV_DELETE:
				log.Debugf("service delete: %s", s)
				delete(client.Services, s)
			case EV_ADD:
				_, ok := client.Services[s]
				if !ok {
					log.Debugf("service add: %s", s)
					client.Services[s] = &serverNode{
						offline:false,
						host:ip,//m.Service.Address,
						port:port,//m.Service.Port,
					}
				}
			default:
				log.Errorf("unknown event: %v", event)
			}
		}))
		members, _, err := w.getMembers()
		if err != nil {
			log.Printf("%+v", err)
		}
		client.lock.Lock()
		for _, m := range members  {
			s := fmt.Sprintf("%v:%v", m.Service.Address, m.Service.Port)
			_, ok := client.Services[s]
			if !ok {
				log.Debugf("1-add : %v", s)
				client.Services[s] = &serverNode{
					offline:false,
					host:m.Service.Address,
					port:m.Service.Port,
				}
			}
		}
		client.lock.Unlock()
		log.Debugf("1-current services list: %+v", client.Services)
		client.getConnects = w.getConnects
		go client.start(wi)
	} else {
		log.Panicf("param error")
	}
	<-wi.c
	return client
}

func OnEventOption(f OnEventFunc) ClientOption{
	return func(client *Client) {
		client.onevent = append(client.onevent, f)
	}
}

func SetServices (ss []string) ClientOption {
	return func(client *Client) {
		for _, s := range ss  {
			temp := strings.Split(s, ":")
			host := temp[0]
			port, _:=strconv.ParseInt(temp[1], 10, 64)
			client.Services[s] = &serverNode{
				offline:false,
				host:host,//m.Service.Address,
				port:int(port),//m.Service.Port,
			}
		}
	}
}

func SetConsulAddress(a string) ClientOption {
	return func(client *Client) {
		client.consulAddress = a
	}
}

// 这里的主题，其实就是 database.table 数据库.表明
// 支持正则，比如test库下面的所有表：test.*
func (client *Client) Subscribe(topics ...string) {
	// 订阅主题
	if client.node == nil {
		log.Errorf("client is not connect")
		return
	}
	for _, t := range topics {
		found := false
		for _, st := range client.topics {
			if st == t {
				found = true
				break
			}
		}
		if !found {
			client.topics = append(client.topics, t)
			clientH := client.setPro(t)
			client.node.conn.Write(clientH)
		}
	}
}

func (client *Client) connect(server *serverNode) {
	log.Debugf("connect to %+v", *server)
	client.lock.Lock()
	defer client.lock.Unlock()
	if client.node != nil && client.node.status & nodeOnline > 0 {
		client.disconnect()
	}
	dns := fmt.Sprintf("%v:%v", server.host, server.port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", dns)
	if err != nil {
		log.Errorf("connect to %+v with error: %+v", *server, err)
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
		for _, t:= range client.topics {
			clientH := client.setPro(t)
			client.node.conn.Write(clientH)
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

func (client *Client) getServer() *serverNode {
	if len(client.Services) <= 0 {
		return nil
	}
	var currentNode *serverNode
	currentNode = nil
	//var min = uint64(0)
	for _, server := range client.Services {
		if server.offline {
			continue
		}
		currentNode = server
		currentNode.connects = client.getConnects(server.host, server.port)
		break
	}
	for _, server := range client.Services {
		if server.offline {
			continue
		}
		server.connects = client.getConnects(server.host, server.port)
		if server.connects < currentNode.connects {
			currentNode = server
		}
	}
	return currentNode
}

func (client *Client) start(wi *wait) {
	client.keepalive()
	var readBuffer [tcpDefaultReadBufferSize]byte
	for {
		server := client.getServer()
		if server == nil {
			time.Sleep(time.Second)
			continue
		}
		client.connect(server)
		if  client.node == nil || client.node.conn == nil || client.node.status & nodeOffline > 0 {
			time.Sleep(time.Second)
			continue
		}
		if !wi.closed {
			close(wi.c)
			wi.closed = true
		}
		if client.status & clientOffline > 0 {
			return
		}
		log.Debugf("====================client start====================")
		for {
			if client.status & clientOffline > 0 {
				return
			}
			size, err := client.node.conn.Read(readBuffer[0:])
			if err != nil || size <= 0 {
				log.Warnf("client read with error: %+v", err)
				client.disconnect()
				server.offline = true
				break
			}
			client.onMessage(readBuffer[:size])
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
			var data map[string]interface{}
			json.Unmarshal(dataB, &data)
			log.Debugf("%+v", data)

			for _, f := range client.onevent {
				f(data)
			}
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