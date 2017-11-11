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

type client_node struct {
	conn *net.Conn
	is_connected bool
	send_queue chan []byte
	send_failure_times int64
}

const (
	TCP_MAX_SEND_QUEUE = 4096
	DEFAULT_TCP_CLIENT_SIZE = 64
	DEFAULT_READ_BUFFER_SIZE = 1024
)

type TcpService struct {
	Ip string
	Port int
	clients []*client_node
	recv_times int64
	send_times int64
	send_failure_times int64
	send_queue chan []byte
	lock *sync.Mutex
}

func NewTcpService(ip string, port int) *TcpService {
	tcp := &TcpService{
		Ip:ip,
		Port:port,
	}

	var con [DEFAULT_TCP_CLIENT_SIZE]*client_node
	tcp.clients = con[:]
	tcp.recv_times = 0
	tcp.send_times = 0
	tcp.send_failure_times = 0
	tcp.send_queue = make(chan []byte, TCP_MAX_SEND_QUEUE)
	tcp.lock = new(sync.Mutex)
	return tcp
}

/**
 * 对外的广播发送接口
 */
func (tcp *TcpService) SendAll(msg []byte) bool {
	if len(tcp.send_queue >= cap(tcp.send_queue)) {
		log.Println("tcp发送缓冲区满...")
		return false
	}
	tcp.send_queue <- msg
	return true
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

func (tcp *TcpService) clientSendService(node *client_node) {
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
	cnode := &client_node{&conn, true, make(chan []byte, TCP_MAX_SEND_QUEUE), 0}
	tcp.clients = append(tcp.clients, cnode)
	tcp.lock.Unlock()


	go tcp.clientSendService(cnode)

	var read_buffer [DEFAULT_READ_BUFFER_SIZE]byte

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

		tcp.onMessage(conn, rbuf)
	}

}

/**
 * 广播服务
 */
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

//收到消息回调函数
func (tcp *TcpService) onMessage(conn net.Conn, msg []byte) {

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