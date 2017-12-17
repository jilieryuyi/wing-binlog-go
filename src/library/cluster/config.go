package cluster

import (
	"net"
	"sync"
	"library/buffer"
	"library/file"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"errors"
	"context"
	"os"
)
var (
	ErrorFileNotFound = errors.New("config file not fount")
	ErrorFileParse = errors.New("parse config error")
)
const (
	TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
	TCP_DEFAULT_CLIENT_SIZE       = 64
	TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
	TCP_RECV_DEFAULT_SIZE         = 4096
	TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
	CLUSTER_NODE_DEFAULT_SIZE     = 4
)

const (
	CMD_APPEND_NODE   = 1
	CMD_POS    = 2
	CMD_CONNECT_FIRST = 3
	CMD_APPEND_NODE_SURE = 4
)

type Cluster struct {
	Listen string      //监听ip，一般为0.0.0.0即可
	Port int           //节点端口
	ServiceIp string   //对外服务ip
	is_down bool       //是否已下线
	client *tcpClient
	server *TcpServer
	nodes []*cluster_node
	nodes_count int
	lock *sync.Mutex
}

type cluster_node struct {
	service_ip string
	port int
	is_enable bool
}

type tcpClient struct {
	dns string
	conn *net.Conn
	isClosed bool
	recvTimes int64
	recvBuf *buffer.WBuffer   //[]byte
	//clientId string           //用来标识一个客户端，随机字符串
	lock *sync.Mutex          // 互斥锁，修改资源时锁定
}

type tcpClientNode struct {
	conn *net.Conn           // 客户端连接进来的资源句柄
	is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
	send_queue chan []byte   // 发送channel
	sendFailureTimes int64 // 发送失败次数
	weight int               // 权重 0 - 100
	recvBuf *buffer.WBuffer  //[]byte          // 读缓冲区
	connect_time int64       // 连接成功的时间戳
	send_times int64         // 发送次数，用来计算负载均衡，如果 mode == 2
}

type TcpServer struct {
	listen string
	port int
	Client *tcpClient
	clients []*tcpClientNode
	clientsCount int
	lock *sync.Mutex          // 互斥锁，修改资源时锁定
	listener *net.Listener
	wg *sync.WaitGroup
	sendFailureTimes int64
	ctx *context.Context
	cacheHandler *os.File
}

type clusterConfig struct{
	Cluster nodeConfig
}

type nodeConfig struct {
	Listen string
	Port int
	//ServiceIp string
}

func getServiceConfig() (*clusterConfig, error) {
	var config clusterConfig
	config_file := file.CurrentPath+"/config/cluster.toml"
	wfile := file.WFile{config_file}
	if !wfile.Exists() {
		log.Errorf("config file %s does not exists", config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(config_file, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &config, nil
}
