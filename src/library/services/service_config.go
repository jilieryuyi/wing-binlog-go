package services

import (
	"errors"
	"sync"
	"library/app"
	"time"
	"net"
)

type Service interface {
	SendAll(table string, data []byte) bool
	SendPos(data []byte)
	Start()
	Close()
	Reload()
	AgentStart(serviceIp string, port int)
	AgentStop()
}

var (
	ErrorFileNotFound = errors.New("config file not found")
	ErrorFileParse    = errors.New("config parse error")
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
	tcpMaxSendQueue               = 10000
	httpMaxSendQueue              = 10000
	tcpDefaultReadBufferSize      = 1024
)

const (
	FlagSetPro = iota
	FlagPing
	FlagControl
	FlagAgent
)

const (
	serviceEnable = 1 << iota
	serviceDisable
	agentStatusOffline
	agentStatusOnline
	agentStatusConnect
	agentStatusDisconnect
)

type httpNodeConfig struct {
	Name   string
	Nodes  []string
	Filter []string
}

type httpGroup struct {
	name   string
	filter []string
	nodes  []*httpNode
}

type HttpService struct {
	Service                                //
	groups           map[string]*httpGroup //
	lock             *sync.Mutex           // 互斥锁，修改资源时锁定
	timeTick         time.Duration         // 故障检测的时间间隔
	ctx              *app.Context
	wg               *sync.WaitGroup
	status           int
}

type HttpConfig struct {
	Enable   bool
	TimeTick time.Duration //故障检测的时间间隔，单位为秒
	Groups   map[string]httpNodeConfig
}

type httpNode struct {
	url              string      // url
	sendQueue        chan string // 发送channel
	lock             *sync.Mutex // 互斥锁，修改资源时锁定
}

const (
	tcpNodeOnline = 1 << iota
	tcpNodeOffline
	tcpNodeIsNormal
	tcpNodeIsAgent
	tcpNodeIsControl
)

type tcpClientNode struct {
	conn             *net.Conn   // 客户端连接进来的资源句柄
	sendQueue        chan []byte // 发送channel
	sendFailureTimes int64       // 发送失败次数
	group            string      // 所属分组
	recvBuf          []byte      // 读缓冲区
	connectTime      int64       // 连接成功的时间戳
	status           int
	wg               *sync.WaitGroup
	ctx              *app.Context
	lock             *sync.Mutex          // 互斥锁，修改资源时锁定
}

type tcpClients []*tcpClientNode

type tcpGroup struct {
	name   string
	filter []string
	nodes  tcpClients
}

type TcpService struct {
	Service
	Ip               string               // 监听ip
	Port             int                  // 监听端口
	sendFailureTimes int64                // 发送失败的次数
	lock             *sync.Mutex
	groups           map[string]*tcpGroup
	ctx              *app.Context
	listener         *net.Listener
	wg               *sync.WaitGroup
	ServiceIp        string
	agents           tcpClients
	status           int
	token            string
	conn             *net.TCPConn
	buffer           []byte
}

type tcpGroupConfig struct {
	Name   string
	Filter []string
}

type TcpConfig struct {
	Listen string `toml:"listen"`
	Port   int    `toml:"port"`
	Enable bool   `toml:"enable"`
	ServiceIp string `toml:"service_ip"`
	Groups map[string]tcpGroupConfig
}

var (
	_ Service = &TcpService{}
	_ Service = &HttpService{}

	packDataTokenError = pack(CMD_AUTH, []byte("token error"))
	packDataTickOk     = pack(CMD_TICK, []byte("ok"))
	packDataSetPro     = pack(CMD_SET_PRO, []byte("ok"))
)


