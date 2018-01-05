package services

import (
	"github.com/BurntSushi/toml"
	"github.com/gorilla/websocket"
	kafka "github.com/segmentio/kafka-go"
	"library/file"
	log "github.com/sirupsen/logrus"
	"errors"
	"sync"
	"net"
	"context"
)

// 标准服务接口
type Service interface {
	// 发送消息
	SendAll(msg []byte) bool
	// 开始服务
	Start()
	Close()
	Reload()
}

type tcpGroupConfig struct { // group node in toml
	Mode int        // "1 broadcast" ##(广播)broadcast or  2 (权重)weight
	Name string     // = "group1"
	Filter []string //
}

type TcpConfig struct {
	Listen string  `toml:"listen"`
	Port   int     `toml:"port"`
	Enable bool    `toml:"enable"`
	Groups map[string]tcpGroupConfig
}

type httpNodeConfig struct {
	Name   string
	Mode   int
	Nodes  [][]string
	Filter []string
}

type websocketClientNode struct {
	conn *websocket.Conn     // 客户端连接进来的资源句柄
	is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
	send_queue chan []byte   // 发送channel
	send_failure_times int64 // 发送失败次数
	mode int                 // broadcast = 1 weight = 2 支持两种方式，广播和权重
	weight int               // 权重 0 - 100
	group string             // 所属分组
	recv_bytes int           // 收到的待处理字节数量
	connect_time int64       // 连接成功的时间戳
	send_times int64         // 发送次数，用来计算负载均衡，如果 mode == 2
}

type WebSocketService struct {
	Service
	Ip string                             // 监听ip
	Port int                              // 监听端口
	recv_times int64                      // 收到消息的次数
	send_times int64                      // 发送消息的次数
	send_failure_times int64              // 发送失败的次数
	lock *sync.Mutex                      // 互斥锁，修改资源时锁定
	groups map[string][]*websocketClientNode // 客户端分组，现在支持两种分组，广播组合负载均衡组
	groups_mode map[string] int           // 分组的模式 1，2 广播还是复载均衡
	groups_filter map[string] []string    // 分组的过滤器
	clients_count int32                   // 成功连接（已经进入分组）的客户端数量
	enable bool
	ctx *context.Context
	wg *sync.WaitGroup
}

type tcpClientNode struct {
	conn *net.Conn          // 客户端连接进来的资源句柄
	isConnected bool        // 是否还连接着 true 表示正常 false表示已断开
	sendQueue chan []byte   // 发送channel
	sendFailureTimes int64  // 发送失败次数
	mode int                // broadcast = 1 weight = 2 支持两种方式，广播和权重
	weight int              // 权重 0 - 100
	group string            // 所属分组
	recvBuf []byte          // 读缓冲区
	recvBytes int           // 收到的待处理字节数量
	connectTime int64       // 连接成功的时间戳
	sendTimes int64         // 发送次数，用来计算负载均衡，如果 mode == 2
}

type tcpGroup struct {
	name   string           //
	mode   int              //
	filter []string         //
	nodes  []*tcpClientNode //
}

type TcpService struct {
	Service
	Ip string                             // 监听ip
	Port int                              // 监听端口
	recvTimes int64                       // 收到消息的次数
	sendTimes int64                       // 发送消息的次数
	sendFailureTimes int64                // 发送失败的次数
	lock *sync.Mutex                      // 互斥锁，修改资源时锁定
	groups map[string]*tcpGroup           //
	enable bool                           //
	ctx *context.Context                  //
	listener *net.Listener                //
	wg *sync.WaitGroup                    //
}

type WKafka struct {
	Service
	writer *kafka.Writer
	is_closed bool
	lock *sync.Mutex
	send_queue chan []byte
	enable bool
	ctx *context.Context
}

type KafkaConfig struct {
	Borkers []string `toml:"brokers"`
	Topic string `toml:"topic"`
	Enable bool `toml:"enable"`
}

var (
	ErrorFileNotFound = errors.New("config file not found")
	ErrorFileParse = errors.New("config parse error")
)

const (
	MODEL_BROADCAST = 1  // 广播
	MODEL_WEIGHT    = 2  // 权重

	CMD_SET_PRO = 1 // 注册客户端操作，加入到指定分组
	CMD_AUTH    = 2 // 认证（暂未使用）
	CMD_OK      = 3 // 正常响应
	CMD_ERROR   = 4 // 错误响应
	CMD_TICK    = 5 // 心跳包
	CMD_EVENT   = 6 // 事件

	TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
	TCP_DEFAULT_CLIENT_SIZE       = 64
	TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
	TCP_RECV_DEFAULT_SIZE         = 4096
	TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096

	HTTP_CACHE_LEN         = 10000
	HTTP_CACHE_BUFFER_SIZE = 4096
)

func getKafkaConfig() (*KafkaConfig, error) {
	var config KafkaConfig
	config_file := file.GetCurrentPath() + "/config/kafka.toml"
	wfile := file.WFile{config_file}
	if !wfile.Exists() {
		log.Errorf("配置文件%s不存在 %s", config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(config_file, &config); err != nil {
		log.Errorf("配置文件解析错误：%+v", err)
		return nil, ErrorFileParse
	}
	return &config, nil
}

func getTcpConfig() (*TcpConfig, error) {
	var tcp_config TcpConfig
	tcp_config_file := file.GetCurrentPath() + "/config/tcp.toml"
	wfile := file.WFile{tcp_config_file}
	if !wfile.Exists() {
		log.Warnf("配置文件%s不存在 %s", tcp_config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(tcp_config_file, &tcp_config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &tcp_config, nil
}

func getHttpConfig() (*HttpConfig, error) {
	var config HttpConfig
	http_config_file := file.GetCurrentPath() + "/config/http.toml"
	wfile := file.WFile{http_config_file}
	if !wfile.Exists() {
		log.Warnf("配置文件%s不存在 %s", http_config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(http_config_file, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	if config.TimeTick <= 0 {
		config.TimeTick = 1
	}
	return &config, nil
}

func getWebsocketConfig() (*TcpConfig, error) {
	var config TcpConfig
	config_file := file.GetCurrentPath() + "/config/websocket.toml"
	wfile := file.WFile{config_file}
	if !wfile.Exists() {
		log.Warnf("配置文件%s不存在 %s", config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(config_file, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &config, nil
}

