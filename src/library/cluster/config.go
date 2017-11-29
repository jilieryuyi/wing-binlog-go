package cluster

import "net"

const (
	TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
	TCP_DEFAULT_CLIENT_SIZE       = 64
	TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
	TCP_RECV_DEFAULT_SIZE         = 4096
	TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
)

const (
	CMD_APPEND_NODE = 1
	CMD_APPEND_NET  = 2
)

type Cluster struct {
	Ip string     //节点ip
	Port int      //节点端口
	next *Cluster //下一个节点
	prev *Cluster //前一个节点
	is_first bool //是否为第一个
	is_last bool   //是否为最后一个
	is_down bool  //是否已下线
	index int
	client *tcp_client
	server *tcp_server
}

type tcp_client struct {
	ip string
	port int
	conn *net.Conn
	is_closed bool
	recv_times int64
	recv_bytes int64
	recv_buf []byte
	client_id string //用来标识一个客户端，随机字符串
}

type tcp_client_node struct {
	conn *net.Conn           // 客户端连接进来的资源句柄
	is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
	send_queue chan []byte   // 发送channel
	send_failure_times int64 // 发送失败次数
	weight int               // 权重 0 - 100
	recv_buf []byte          // 读缓冲区
	recv_bytes int           // 收到的待处理字节数量
	connect_time int64       // 连接成功的时间戳
	send_times int64         // 发送次数，用来计算负载均衡，如果 mode == 2
}

type tcp_server struct {
	listen string
	port int
	client *tcp_client
	clients []*tcp_client_node
}
