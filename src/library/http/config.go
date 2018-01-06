package http

import (
	"errors"
	"github.com/BurntSushi/toml"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"library/file"
	"net/http"
	"sync"
)

var (
	ErrorFileNotFound = errors.New("没找到配置文件")
	ErrorFileParse    = errors.New("配置文件读取错误")
	ErrorHttpStatus   = errors.New("错误的状态码")
)

type tcpConfig struct {
	Listen string
	Port   int
}

type websocketConfig struct {
	Listen string
	Port   int
}

type httpServiceConfig struct {
	Http      tcpConfig
	Websocket websocketConfig
}

type HttpServer struct {
	Path        string // web路径 当前路径/web
	Ip          string // 监听ip 0.0.0.0
	Port        int    // 9989
	ws          *WebSocketService
	httpHandler http.Handler
}

type OnLineUser struct {
	Name         string
	Password     string
	LastPostTime int64
}

type websocketClientNode struct {
	conn               *websocket.Conn // 客户端连接进来的资源句柄
	is_connected       bool            // 是否还连接着 true 表示正常 false表示已断开
	send_queue         chan []byte     // 发送channel
	send_failure_times int64           // 发送失败次数
	recv_bytes         int             // 收到的待处理字节数量
	connect_time       int64           // 连接成功的时间戳
	send_times         int64           // 发送次数，用来计算负载均衡，如果 mode == 2
}

type WebSocketService struct {
	Ip                 string      // 监听ip
	Port               int         // 监听端口
	recv_times         int64       // 收到消息的次数
	send_times         int64       // 发送消息的次数
	send_failure_times int64       // 发送失败的次数
	send_queue         chan []byte // 发送队列-广播
	lock               *sync.Mutex // 互斥锁，修改资源时锁定
	clients_count      int32       // 成功连接（已经进入分组）的客户端数量
	clients            map[string]*websocketClientNode
}

const (
	CMD_SET_PRO = 1 // 注册客户端操作，加入到指定分组
	CMD_AUTH    = 2 // 认证（暂未使用）
	CMD_OK      = 3 // 正常响应
	CMD_ERROR   = 4 // 错误响应
	CMD_TICK    = 5 // 心跳包
	CMD_EVENT   = 6 // 事件
	CMD_CONNECT = 7
	CMD_RELOGIN = 8
)

const (
	TCP_MAX_SEND_QUEUE            = 128 //100万缓冲区
	TCP_DEFAULT_CLIENT_SIZE       = 64
	TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
	TCP_RECV_DEFAULT_SIZE         = 4096
	TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
	HTTP_POST_TIMEOUT             = 3   //3秒超时
	DEFAULT_LOGIN_TIMEOUT         = 600 //600秒不活动，登录超时
)

func getServiceConfig() (*httpServiceConfig, error) {
	var config httpServiceConfig
	config_file := file.GetCurrentPath() + "/config/admin.toml"
	wfile := file.WFile{config_file}
	if !wfile.Exists() {
		log.Printf("配置文件%s不存在", config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(config_file, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &config, nil
}
