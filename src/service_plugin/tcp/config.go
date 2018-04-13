package tcp

import (
	"sync"
	"library/app"
	"net"
	"library/services"
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
)

const (
	serviceEnable = 1 << iota
	serviceClosed
)

const (
	tcpNodeOnline = 1 << iota
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
	onclose []NodeFunc
	onpro SetProFunc
}

type NodeFunc func(n *tcpClientNode)
type SetProFunc func(n *tcpClientNode, groupName string) bool
type NodeOption func(n *tcpClientNode)

type tcpClients []*tcpClientNode
//type tcpGroups map[string]*tcpGroup

type tcpGroup struct {
	name   string
	filter []string
	nodes  tcpClients
	lock *sync.Mutex
}

type TcpService struct {
	services.Service
	Ip               string               // 监听ip
	Port             int                  // 监听端口
	lock             *sync.Mutex
	statusLock       *sync.Mutex
	//groups           tcpGroups
	ctx              *app.Context
	listener         *net.Listener
	wg               *sync.WaitGroup
	ServiceIp        string
	status           int
	token            string
	conn             *net.TCPConn
	buffer           []byte

	sendAll []SendAllFunc
	sendRaw []SendRawFunc
	onConnect []OnConnectFunc
	onClose []CloseFunc
	onKeepalive []KeepaliveFunc
	reload []ReloadFunc
}

var (
	_ services.Service = &TcpService{}
	packDataTickOk     = services.Pack(CMD_TICK, []byte("ok"))
	packDataSetPro     = services.Pack(CMD_SET_PRO, []byte("ok"))
)

type TcpServiceOption func(service *TcpService)
type SendAllFunc func(table string, data []byte) bool
type SendRawFunc func(msg []byte)
type OnConnectFunc func(conn *net.Conn)
type CloseFunc func()
type KeepaliveFunc func(data []byte)
type ReloadFunc func()



