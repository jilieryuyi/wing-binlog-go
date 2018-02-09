package services

import (
	"errors"
	"sync"
	"library/app"
	"time"
	"net"
)

type Service interface {
	SendAll(data map[string] interface{}) bool
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
	CMD_SET_PRO = 1 // 注册客户端操作，加入到指定分组
	CMD_AUTH    = 2 // 认证（暂未使用）
	CMD_OK      = 3 // 正常响应
	CMD_ERROR   = 4 // 错误响应
	CMD_TICK    = 5 // 心跳包
	CMD_EVENT   = 6 // 事件
	CMD_AGENT   = 7

	TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
	TCP_DEFAULT_CLIENT_SIZE       = 64
	tcpDefaultReadBufferSize      = 1024
	tcpRecviveDefaultSize         = 4096

	HTTP_CACHE_LEN         = 10000
)


type httpNodeConfig struct {
	Name   string
	Nodes  []string
	Filter []string
}

type httpGroup struct {
	name   string      //
	filter []string    //
	nodes  []*httpNode //
}

type HttpService struct {
	Service                                //
	groups           map[string]*httpGroup //
	lock             *sync.Mutex           // 互斥锁，修改资源时锁定
	sendFailureTimes int64                 // 发送失败次数
	enable           bool                  //
	timeTick         time.Duration         // 故障检测的时间间隔
	ctx              *app.Context//*context.Context      //
	wg               *sync.WaitGroup       //
}

type HttpConfig struct {
	Enable   bool
	TimeTick time.Duration //故障检测的时间间隔，单位为秒
	Groups   map[string]httpNodeConfig
}

type httpNode struct {
	url              string      // url
	sendQueue        chan string // 发送channel
	sendTimes        int64       // 发送次数
	sendFailureTimes int64       // 发送失败次数
	isDown           bool        // 是否因为故障下线的节点
	failureTimesFlag int32       // 发送失败次数，用于配合last_error_time检测故障，故障定义为：连续三次发生错误和返回错误
	lock             *sync.Mutex // 互斥锁，修改资源时锁定
	cache            [][]byte
	cacheIndex       int
	cacheIsInit      bool
	cacheFull        bool
	errorCheckTimes  int64
}

type tcpClientNode struct {
	conn             *net.Conn   // 客户端连接进来的资源句柄
	isConnected      bool        // 是否还连接着 true 表示正常 false表示已断开
	sendQueue        chan []byte // 发送channel
	sendFailureTimes int64       // 发送失败次数
	group            string      // 所属分组
	recvBuf          []byte      // 读缓冲区
	recvBytes        int         // 收到的待处理字节数量
	connectTime      int64       // 连接成功的时间戳
	sendTimes        int64       // 发送次数，用来计算负载均衡，如果 mode == 2
	isAgent          bool
}

type tcpGroup struct {
	name   string
	filter []string
	nodes  []*tcpClientNode
}

type TcpService struct {
	Service
	Ip               string               // 监听ip
	Port             int                  // 监听端口
	recvTimes        int64                // 收到消息的次数
	sendTimes        int64                // 发送消息的次数
	sendFailureTimes int64                // 发送失败的次数
	lock             *sync.Mutex          // 互斥锁，修改资源时锁定
	groups           map[string]*tcpGroup //
	enable           bool                 //
	ctx              *app.Context//*context.Context     //
	listener         *net.Listener        //
	wg               *sync.WaitGroup      //
	Agent            *Agent
	ServiceIp        string
	Agents           []*tcpClientNode
	sendAllChan1 chan map[string] interface{}
	sendAllChan2 chan []byte
}

type tcpGroupConfig struct { // group node in toml
	Name   string   // = "group1"
	Filter []string //
}

type TcpConfig struct {
	Listen string `toml:"listen"`
	Port   int    `toml:"port"`
	Enable bool   `toml:"enable"`
	ServiceIp string `toml:"service_ip"`
	Groups map[string]tcpGroupConfig
}


