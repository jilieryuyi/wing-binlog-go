package agent

import (
	"sync"
	"library/app"
	"net"
	//"library/services"
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
	agentStatusOnline
	agentStatusConnect
)

const (
	serviceIsLeader = 1 << iota
)

const (
	tcpNodeOnline = 1 << iota
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
type tcpGroups map[string]*tcpGroup

type tcpGroup struct {
	name   string
	filter []string
	nodes  tcpClients
	lock *sync.Mutex
}

type TcpService struct {
	Address          string               // 监听ip
	lock             *sync.Mutex
	statusLock       *sync.Mutex
	ctx              *app.Context
	listener         *net.Listener
	wg               *sync.WaitGroup
	agents           tcpClients
	status           int
	conn             *net.TCPConn
	buffer           []byte
	service *Service
	client *AgentClient
	watch *ConsulWatcher
}
type OnPosFunc func(r []byte)
type AgentServerOption func(s *TcpService)
var (
	//_ services.Service = &TcpService{}
	packDataTokenError = pack(CMD_AUTH, []byte("token error"))
	packDataTickOk     = pack(CMD_TICK, []byte("ok"))
	packDataSetPro     = pack(CMD_SET_PRO, []byte("ok"))
)


