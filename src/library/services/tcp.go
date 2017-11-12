package services

import (
	"fmt"
	"net"
	"log"
	//"runtime"
	"time"
	"sync/atomic"
	"sync"
)

const (
	MODEL_BROADCAST = 1
	MODEL_WEIGHT = 2
)

const (
	CMD_SET_PRO = 1 //<< iota
	CMD_AUTH = 2
	CMD_OK = 3
	CMD_ERROR = 4
	CMD_TICK = 5
	CMD_EVENT = 6
)

const (
	TCP_MAX_SEND_QUEUE           = 1000000
	TCP_DEFAULT_CLIENT_SIZE      = 64
	TCP_DEFAULT_READ_BUFFER_SIZE = 1024
	TCP_RECV_DEFAULT_SIZE        = 4096
)

type tcp_client_node struct {
	conn *net.Conn           // 客户端连接进来的资源句柄
	is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
	send_queue chan []byte   // 发送channel
	send_failure_times int64 // 发送失败次数
	mode int                 // broadcast = 1 weight = 2 支持两种方式，广播和权重
	weight int               // 权重 0 - 100
	group string             // 所属分组
	recv_buf []byte          // 读缓冲区
	recv_bytes int           // 收到的待处理字节数量
	connect_time int64       // 连接成功的时间戳
	send_times int64         // 发送次数，用来计算负载均衡，如果 mode == 2
}

type TcpService struct {
	Ip string                             // 监听ip
	Port int                              // 监听端口
	recv_times int64                      // 收到消息的次数
	send_times int64                      // 发送消息的次数
	send_failure_times int64              // 发送失败的次数
	send_queue chan []byte                // 发送队列-广播
	lock *sync.Mutex                      // 互斥锁，修改资源时锁定
	groups map[string][]*tcp_client_node  // 客户端分组，现在支持两种分组，广播组合负载均衡组
	groups_mode map[string] int           // 分组的模式 1，2 广播还是复载均衡
	clients_count int32                   // 成功连接（已经进入分组）的客户端数量
}

func NewTcpService(ip string, port int, config *TcpConfig) *TcpService {
	tcp := &TcpService{
		Ip:ip,
		Port:port,
		clients_count:int32(0),
		lock:new(sync.Mutex),
	}

	tcp.send_queue  = make(chan []byte, TCP_MAX_SEND_QUEUE)
	tcp.groups      = make(map[string][]*tcp_client_node)
	tcp.groups_mode = make(map[string] int)

	for _, v := range config.Groups {
		var con [TCP_DEFAULT_CLIENT_SIZE]*tcp_client_node
		tcp.groups[v.Name]      = con[:0]
		tcp.groups_mode[v.Name] = v.Mode
	}

	tcp.recv_times         = 0
	tcp.send_times         = 0
	tcp.send_failure_times = 0
	return tcp
}

// 对外的广播发送接口
func (tcp *TcpService) SendAll(msg []byte) bool {
	cc := atomic.LoadInt32(&tcp.clients_count)
	if cc <= 0 {
		return false
	}
	if len(tcp.send_queue) >= cap(tcp.send_queue) {
		log.Println("tcp发送缓冲区满...")
		return false
	}
	tcp.send_queue <- tcp.pack(CMD_EVENT, string(msg))
	return true
}

// 广播服务
func (tcp *TcpService) broadcast() {
	to := time.NewTimer(time.Second*1)
	//cpu := runtime.NumCPU()
	//for i := 0; i < cpu; i ++
	//{
		tou := to
		//go func()
		//{

			for {
				select {
				case  msg := <-tcp.send_queue:
					tcp.lock.Lock()
					log.Println(tcp.groups)
					//分组
					for group_name, clients := range tcp.groups {

						if len(clients) <= 0 {
							continue
						}

						mode := tcp.groups_mode[group_name]

						if mode != MODEL_WEIGHT {
							for _, conn := range clients {
								conn.send_queue <- msg
							}
						} else {
							log.Println(clients)
							// todo 根据已经send_times的次数负载均衡
							target := clients[0]
							//将发送次数/权重 作为负载基数，每次选择最小的发送
							js := atomic.LoadInt64(&target.send_times)/int64(target.weight)

							for _, conn := range clients {
								stimes := atomic.LoadInt64(&conn.send_times)
								//conn.send_queue <- msg
								if stimes == 0 {
									//优先发送没有发过的
									target = conn
									break
								}

								_js := stimes/int64(conn.weight)
								if _js < js {
									js = _js
									target = conn
								}
							}

							target.send_queue <- msg
						}
					}
					tcp.lock.Unlock()
				case <-tou.C://time.After(time.Second*3):
				}
			}
		//} ()
	//}
}

// 打包tcp响应包 格式为 [包长度-2字节，大端序][指令-2字节][内容]
func (tcp *TcpService) pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m)
	r := make([]byte, l + 6)

	cl := l + 2
	log.Println("发送消息：", l, msg)

	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 32)

	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], m)

	log.Println(r)

	return r
}


func (tcp *TcpService) onClose(conn *tcp_client_node) {
	if conn.group == "" {
		conn.is_connected = false
		return
	}
	//移除conn
	//查实查找位置
	tcp.lock.Lock()
	for index, con := range tcp.groups[conn.group] {
		if con.conn == conn.conn {
			con.is_connected = false
			tcp.groups[conn.group] = append(tcp.groups[conn.group][:index], tcp.groups[conn.group][index+1:]...)
			break
		}
	}
	tcp.lock.Unlock()
	atomic.AddInt32(&tcp.clients_count, int32(-1))
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
			atomic.AddInt64(&node.send_times, int64(1))

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

	//tcp.lock.Lock()
	cnode := &tcp_client_node{
		conn:&conn,
		is_connected:true,
		send_queue:make(chan []byte, TCP_MAX_SEND_QUEUE),
		send_failure_times:0,
		weight:0,
		mode: MODEL_BROADCAST,
		connect_time:time.Now().Unix(),
		send_times:int64(0),
	}
	cnode.recv_buf = make([]byte, TCP_RECV_DEFAULT_SIZE)
	cnode.recv_bytes = 0
	cnode.group = ""
	//tcp.clients = append(tcp.clients, cnode)
	//tcp.lock.Unlock()


	go tcp.clientSendService(cnode)


	var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte//, )// [TCP_DEFAULT_READ_BUFFER_SIZE]byte

	//conn.SetReadDeadline(time.Now().Add(time.Second*3))
	//conn.SetDeadline(time.Now().Add(time.Second*3))
	for {
		buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据
		for k,_:= range buf {
			buf[k] = byte(0)
		}
		size, err := conn.Read(buf)

		if err != nil {
			log.Println(conn.RemoteAddr().String(), "连接发生错误: ", err)
			tcp.onClose(cnode);
			conn.Close();
			return
		}

		log.Println("收到消息",size,"字节：", string(buf))
		atomic.AddInt64(&tcp.recv_times, int64(1))

		cnode.recv_bytes += size
		tcp.onMessage(cnode, buf, size)
	}

}
// 收到消息回调函数
func (tcp *TcpService) onMessage(conn *tcp_client_node, msg []byte, size int) {
	//log.Println("recv_buf: ", len(msg), msg)
	conn.recv_buf = append(conn.recv_buf[:conn.recv_bytes - size], msg[0:size]...)

	for {
		clen := len(conn.recv_buf)
		if clen < 6 {
			return
		} else if clen > TCP_RECV_DEFAULT_SIZE {
			// 清除所有的读缓存，防止发送的脏数据不断的累计
			conn.recv_buf = make([]byte, TCP_RECV_DEFAULT_SIZE)
			log.Println("新建缓冲区")
			return
		}

		//4字节长度
		content_len := int(conn.recv_buf[0]) +
			int(conn.recv_buf[1] << 8) +
			int(conn.recv_buf[2] << 16) +
			int(conn.recv_buf[3] << 32)

		//2字节 command
		cmd := int(conn.recv_buf[4]) + int(conn.recv_buf[5] << 8)

		log.Println("content：", conn.recv_buf)
		log.Println("content_len：", content_len)
		log.Println("cmd：", cmd)
		switch cmd {
		case CMD_SET_PRO:
			if len(conn.recv_buf) < 10 {
				return
			}
			//4字节 weight
			weight := int(conn.recv_buf[6]) +
				int(conn.recv_buf[7] << 8) +
				int(conn.recv_buf[8] << 16) +
				int(conn.recv_buf[9] << 32)

			log.Println("weight：", weight)
			if weight < 0 || weight > 100 {
				conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的权重值：%d，请设置为0-100之间", weight))
				return
			}

			//内容长度+4字节的前缀（存放内容长度的数值）
			group := string(conn.recv_buf[10:content_len + 4])
			log.Println("group：", group)

			is_find := false
			for g, _ := range tcp.groups {
				log.Println(g, len(g), ">" + g + "<", len(group), ">" + group + "<")
				if g == group {
					is_find = true
					break
				}
			}
			if !is_find {
				conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("组不存在：%s", group))
				return
			}

			//(*conn.conn).SetReadDeadline(time.Time{})
			conn.send_queue <- tcp.pack(CMD_SET_PRO, "ok")

			tcp.lock.Lock()
			conn.group = group
			conn.mode = tcp.groups_mode[group]
			conn.weight = weight

			log.Println(len(tcp.groups[group]))
			tcp.groups[group] = append(tcp.groups[group], conn)
			log.Println(len(tcp.groups[group]), tcp.groups[group])

			if conn.mode == MODEL_WEIGHT {
				//weight 合理性格式化，保证所有的weight的和是100
				all_weight := 0
				for _, _conn := range tcp.groups[group] {
					w := _conn.weight
					if w <= 0 {
						w = 100
					}
					all_weight += w
				}

				gl := len(tcp.groups[group])
				yg := 0
				for k, _conn := range tcp.groups[group] {
					if k == gl - 1 {
						_conn.weight = 100 - yg
					} else {
						_conn.weight = int(_conn.weight * 100 / all_weight)
						yg += _conn.weight
					}
				}
			}
			atomic.AddInt32(&tcp.clients_count, int32(1))
			tcp.lock.Unlock()

		case CMD_TICK:
			log.Println("收到心跳消息")
			conn.send_queue <- tcp.pack(CMD_OK, "ok")
		//心跳包
		default:
			conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
		}

		//数据移动
		log.Println(content_len + 4, conn.recv_bytes)
		conn.recv_buf = append(conn.recv_buf[:0], conn.recv_buf[content_len + 4:conn.recv_bytes]...)
		conn.recv_bytes = conn.recv_bytes - content_len - 4

		log.Println("移动后的数据：", conn.recv_bytes, len(conn.recv_buf), string(conn.recv_buf))
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
			//for _, c := range tcp.clients {
			//	tcp.onClose(c.conn)
			//}
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
