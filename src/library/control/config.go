package control

import (
	"sync"
	"library/app"
	"net"
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
	tcpNodeOnline = 1 << iota
	tcpNodeIsNormal
	tcpNodeIsAgent
	tcpNodeIsControl
)

type TcpClientNode struct {
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

type TcpService struct {
	Address          string
	lock             *sync.Mutex
	ctx              *app.Context
	listener         *net.Listener
	wg               *sync.WaitGroup
	token            string
	conn             *net.TCPConn
	buffer           []byte
	showmember ShowMemberFunc
	reload ReloadFunc
	stop StopFunc
}

var (
	packDataTickOk     = pack(CMD_TICK, []byte("ok"))
)
type ShowMemberFunc func() string
type ReloadFunc func(service string)
type StopFunc func()
type ControlOption func(tcp *TcpService)


