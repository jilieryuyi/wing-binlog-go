package services

import (
	"fmt"
	"net"
	"log"
	"runtime"
	"time"
	"sync/atomic"
	"sync"
)

const (
	MODEL_BROADCAST = 1
	MODEL_WEIGHT = 2
)

const (
	CMD_SET_PRO = 1 << iota
	CMD_AUTH
)

const (
	CMD_OK    = 1
	CMD_ERROR = 2
)


type tcp_client_node struct {
	conn *net.Conn
	is_connected bool
	send_queue chan []byte
	send_failure_times int64
	mode int //broadcast = 1 weight = 2 支持两种方式，广播和权重
	weight int
	group string
	recv_buf []byte
}

const (
	TCP_MAX_SEND_QUEUE = 4096
	TCP_DEFAULT_CLIENT_SIZE = 64
	TCP_DEFAULT_READ_BUFFER_SIZE = 1024
	TCP_RECV_DEFAULT_SIZE = 4096
)

type TcpService struct {
	Ip string
	Port int
	clients []*tcp_client_node
	recv_times int64
	send_times int64
	send_failure_times int64
	send_queue chan []byte
	lock *sync.Mutex
	groups []string
	config *TcpConfig
}

func NewTcpService(ip string, port int, config *TcpConfig) *TcpService {

	l := len(config.Groups)

	tcp := &TcpService{
		Ip:ip,
		Port:port,
		config:config,
	}

	tcp.groups = make([]string, l)
	i := 0
	for _, v := range config.Groups {
		tcp.groups[i] = v.Name
		i++
	}

	var con [TCP_DEFAULT_CLIENT_SIZE]*tcp_client_node
	tcp.clients = con[:0]
	tcp.recv_times = 0
	tcp.send_times = 0
	tcp.send_failure_times = 0
	tcp.send_queue = make(chan []byte, TCP_MAX_SEND_QUEUE)
	tcp.lock = new(sync.Mutex)
	return tcp
}

// 对外的广播发送接口
func (tcp *TcpService) SendAll(msg []byte) bool {
	clen := len(tcp.clients)
	if clen <=0 {
		return false
	}
	if len(tcp.send_queue) >= cap(tcp.send_queue) {
		log.Println("tcp发送缓冲区满...", clen)
		return false
	}
	tcp.send_queue <- msg
	return true
}

// 广播服务
func (tcp *TcpService) broadcast() {
	to := time.NewTimer(time.Second*1)
	cpu := runtime.NumCPU()
	for i := 0; i < cpu; i ++ {
		tou := to
		go func() {
			for {
				select {
				case  msg := <-tcp.send_queue:
					for _, conn := range tcp.clients {
						conn.send_queue <- msg
					}
				case <-tou.C://time.After(time.Second*3):
				}
			}
		} ()
	}
}

// 打包tcp响应包 格式为 [包长度-2字节，大端序][指令-2字节][内容]
func (tcp *TcpService) pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m) + 2
	r := make([]byte, l + 2)
	r[0] = byte(l)
	r[1] = byte(l >> 8)
	r[3] = byte(cmd)
	r[4] = byte(cmd >> 8)
	copy(r[4:], m)
	return r
}

// 收到消息回调函数
func (tcp *TcpService) onMessage(conn *tcp_client_node, msg []byte) {
	conn.recv_buf = append(conn.recv_buf, msg...)
	if len(conn.recv_buf) < 10 {
		return
	} else if len(conn.recv_buf) > TCP_RECV_DEFAULT_SIZE {
		//清除所有的读缓存，防止发送的脏数据不断的累计
		var nb [TCP_RECV_DEFAULT_SIZE]byte
		conn.recv_buf = nb[:0]
		return
	}

	//2字节长度
	content_len := int(msg[0])+ int(msg[1] << 8)
	//2字节 command
	cmd := int(msg[2])+ int(msg[3] << 8)

	switch cmd {
	case CMD_SET_PRO:
		//2字节 mode
		mode := int(msg[4])+ int(msg[5] << 8)
		//4字节 weight
		weight := int(msg[6])      +
			      int(msg[7] << 8) +
			      int(msg[8] << 16)+
			      int(msg[9] << 32)

		if mode != MODEL_BROADCAST && mode != MODEL_WEIGHT {
			conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的模式：%d", mode))
		} else {

			if weight < 0 || weight > 100 {
				conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的权重值：%d，请设置为0-100之间", weight))
				return
			}

			group := string(conn.recv_buf[9:content_len])
			is_find := false
			for _, g := range tcp.groups {
				if g == group {
					is_find = true
					break
				}
			}
			if !is_find {
				conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("组不存在：%s", group))
				return
			}

			conn.group = group
			conn.mode = mode
			conn.weight = weight
			//数据移动
			copy(conn.recv_buf[:0], conn.recv_buf[content_len:])
			conn.send_queue <- tcp.pack(CMD_OK, "ok")
		}
	default:
		conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
	}
}

func (tcp *TcpService) onClose(conn *net.Conn) {
	//移除conn
	//查实查找位置
	tcp.lock.Lock()
	for index, con := range tcp.clients {
		if con.conn == conn {
			con.is_connected = false
			tcp.clients = append(tcp.clients[:index], tcp.clients[index+1:]...)
			break
		}
	}
	tcp.lock.Unlock()
}



func (tcp *TcpService) clientSendService(node *tcp_client_node) {
	to := time.NewTimer(time.Second*1)

	for {
		if !node.is_connected {
			break
		}

		select {
		case  msg := <-node.send_queue:
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second*1))
			size, err := (*node.conn).Write(msg)
			if (size <= 0 || err != nil) {
				atomic.AddInt64(&tcp.send_failure_times, int64(1))
				atomic.AddInt64(&node.send_failure_times, int64(1))

				log.Println("失败次数：", tcp.send_failure_times, node.conn, node.send_failure_times)
			}
		case <-to.C://time.After(time.Second*3):
			//log.Println("发送超时...", tcp)
		}
	}
}

func (tcp *TcpService) onConnect(conn net.Conn) {
	log.Println("新的连接：",conn.RemoteAddr().String())

	tcp.lock.Lock()
	cnode := &tcp_client_node{
		conn:&conn,
		is_connected:true,
		send_queue:make(chan []byte, TCP_MAX_SEND_QUEUE),
		send_failure_times:0,
		weight:0,
		mode: MODEL_BROADCAST,
	}
	var b [TCP_RECV_DEFAULT_SIZE]byte
	cnode.recv_buf = b[:0]

	/*
	is_connected bool
	send_queue chan []byte
	send_failure_times int64
	mode string //broadcast weight 支持两种方式，广播和权重
	weight int
	*/

	tcp.clients = append(tcp.clients, cnode)
	tcp.lock.Unlock()


	go tcp.clientSendService(cnode)

	var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte

	for {
		rbuf := read_buffer[:0]
		size, err := conn.Read(rbuf)

		if err != nil {
			log.Println(conn.RemoteAddr().String(), "连接发生错误: ", err)
			tcp.onClose(&conn);
			conn.Close();
			return
		}

		log.Println("收到消息",size,"字节：", string(rbuf))
		atomic.AddInt64(&tcp.recv_times, int64(1))

		tcp.onMessage(cnode, rbuf)
	}

}

func (tcp *TcpService) Start() {

	go tcp.broadcast()

	go func() {
		//建立socket，监听端口
		dns := fmt.Sprintf("%s:%d", tcp.Ip, tcp.Port)
		listen, err := net.Listen("tcp", dns)

		if err != nil {
			log.Println(err)
			return
		}

		defer func() {
			listen.Close();
			close(tcp.send_queue)
			for _, c := range tcp.clients {
				tcp.onClose(c.conn)
			}
		}()

		log.Println("等待新的连接...")

		for {
			conn, err := listen.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			go tcp.onConnect(conn)
		}
	} ()
}
