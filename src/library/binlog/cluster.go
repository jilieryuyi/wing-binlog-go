package binlog

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"library/buffer"
	"library/file"
	"context"
	"github.com/BurntSushi/toml"
	"net"
	"os"
)

//
//type Cluster struct {
//	Listen string      //监听ip，一般为0.0.0.0即可
//	Port int           //节点端口
//	ServiceIp string   //对外服务ip
//	is_down bool       //是否已下线
//	client *tcpClient
//	server *TcpServer
//	nodes []*cluster_node
//	nodes_count int
//	lock *sync.Mutex
//}

//type cluster_node struct {
//	service_ip string
//	port int
//	is_enable bool
//}

type tcpClient struct {
	dns string
	conn *net.Conn
	isClosed bool
	recvTimes int64
	recvBuf *buffer.WBuffer   //[]byte
	//clientId string           //用来标识一个客户端，随机字符串
	lock *sync.Mutex          // 互斥锁，修改资源时锁定
	binlog *Binlog
	ServiceIp string
	ServicePort int
}

type tcpClientNode struct {
	conn *net.Conn           // 客户端连接进来的资源句柄
	is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
	send_queue chan []byte   // 发送channel
	sendFailureTimes int64   // 发送失败次数
	weight int               // 权重 0 - 100
	recvBuf *buffer.WBuffer  //[]byte          // 读缓冲区
	connect_time int64       // 连接成功的时间戳
	send_times int64         // 发送次数，用来计算负载均衡，如果 mode == 2
	ServiceDns string        // 节点服务器的服务 ip:port
}

type TcpServer struct {
	listen string
	port int
	Client *tcpClient
	clients []*tcpClientNode  // 所有的集群几点服务器
	clientsCount int
	lock *sync.Mutex          // 互斥锁，修改资源时锁定
	listener *net.Listener
	wg *sync.WaitGroup
	sendFailureTimes int64
	ctx *context.Context
	binlog *Binlog
	ServiceIp string
	cacheHandler *os.File
}

type clusterConfig struct{
	Cluster nodeConfig
}

type nodeConfig struct {
	Listen string
	Port int
	ServiceIp string
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


func NewCluster(ctx *context.Context, binlog *Binlog) *TcpServer {
	config, _:= getServiceConfig()
	log.Infof("cluster server初始化 ...............")
	log.Infof("listen: %s:%d", config.Cluster.Listen, config.Cluster.Port)

	server := &TcpServer{
		listen : config.Cluster.Listen,
		port : config.Cluster.Port,
		lock : new(sync.Mutex),
		clientsCount : 0,
		listener : nil,
		clients:make([]*tcpClientNode, 4),
		wg : new(sync.WaitGroup),
		sendFailureTimes : int64(0),
		ctx : ctx,
		binlog : binlog,
		ServiceIp : config.Cluster.ServiceIp,
	}

	server.Client = &tcpClient{
		isClosed : true,
		recvTimes : int64(0),
		recvBuf : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),
		lock : new(sync.Mutex),
		binlog : binlog,
		ServiceIp : config.Cluster.ServiceIp,
		ServicePort : config.Cluster.Port,
	}


	// 初始化缓存文件句柄
	cache := file.CurrentPath +"/cache/nodes.list"
	dir   := file.WPath{cache}
	dir    = file.WPath{dir.GetParent()}

	dir.Mkdir()
	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC | os.O_TRUNC
	var err error
	server.cacheHandler, err = os.OpenFile(cache, flag , 0755)
	if err != nil {
		log.Panicf("binlog服务，打开缓存文件错误：%s, %+v", cache, err)
	}

	return server
}