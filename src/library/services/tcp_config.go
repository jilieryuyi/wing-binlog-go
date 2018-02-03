package services

import (
	"library/file"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"sync"
	"net"
	"context"
	"library/path"
)

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
	ctx              *context.Context     //
	listener         *net.Listener        //
	wg               *sync.WaitGroup      //
	Agent            *Agent
	//Drive            cluster.Cluster
	ServiceIp        string
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

func getTcpConfig() (*TcpConfig, error) {
	var tcp_config TcpConfig
	tcp_config_file := path.CurrentPath + "/config/tcp.toml"
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