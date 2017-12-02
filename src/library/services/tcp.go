package services

import (
	"fmt"
	"net"
	//log "library/log"
	log "github.com/sirupsen/logrus"
	"time"
	"sync/atomic"
	"sync"
	"regexp"
)

type tcp_client_node struct {
	conn *net.Conn           // 客户端连接进来的资源句柄
	is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
	send_queue chan []byte   // 发送channel
	send_failure_times int64 // 发送失败次数
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
	groups_filter map[string] []string    // 分组的过滤器
	clients_count int32                   // 成功连接（已经进入分组）的客户端数量
}

func NewTcpService(config *TcpConfig) *TcpService {
	//tcp_config.Tcp.Listen, tcp_config.Tcp.Port
	tcp := &TcpService {
		Ip                 : config.Tcp.Listen,
		Port               : config.Tcp.Port,
		clients_count      : int32(0),
		lock               : new(sync.Mutex),
		send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
		groups             : make(map[string][]*tcp_client_node),
		groups_filter      : make(map[string] []string),
		recv_times         : 0,
		send_times         : 0,
		send_failure_times : 0,
	}

	for _, v := range config.Groups {
		flen := len(v.Filter)
		var con [TCP_DEFAULT_CLIENT_SIZE]*tcp_client_node
		tcp.groups[v.Name]      = con[:0]
		tcp.groups_filter[v.Name] = make([]string, flen)
		tcp.groups_filter[v.Name] = append(tcp.groups_filter[v.Name][:0], v.Filter...)
	}

	return tcp
}

// 对外的广播发送接口
func (tcp *TcpService) SendAll(msg []byte) bool {
	log.Println("tcp-sendall")
	cc := atomic.LoadInt32(&tcp.clients_count)
	if cc <= 0 {
		log.Println("tcp-sendall-clients_count 0")
		return false
	}
	if len(tcp.send_queue) >= cap(tcp.send_queue) {
		log.Println("tcp发送缓冲区满...")
		return false
	}

	table_len := int(msg[0]) + int(msg[1] << 8);
	tcp.send_queue <- tcp.pack2(CMD_EVENT, msg[table_len+2:], msg[:table_len+2])
	return true
}

// 广播服务
func (tcp *TcpService) broadcast() {
	to := time.NewTimer(time.Second*1)
	for {
		select {
		case  msg := <-tcp.send_queue:
			log.Println("tcp广播消息")
			tcp.lock.Lock()
			for group_name, clients := range tcp.groups {
				// 如果分组里面没有客户端连接，跳过
				if len(clients) <= 0 {
					continue
				}
				// 分组的模式
				filter := tcp.groups_filter[group_name]
				flen   := len(filter)
				//2字节长度
				table_len := int(msg[0]) + int(msg[1] << 8);
				table     := string(msg[2:table_len+2])

				log.Println("tcp数据表：", table_len, table)
				log.Println(msg)
				//log.Println(filter)

				if flen > 0 {
					is_match := false
					for _, f := range filter {
						match, err := regexp.MatchString(f, table)
						if err != nil {
							continue
						}
						if match {
							is_match = true
						}
					}
					if !is_match {
						continue
					}
				}

				for _, conn := range clients {
					if !conn.is_connected {
						continue
					}
					log.Println("发送广播消息")
					conn.send_queue <- msg[table_len+2:]
				}
			}
			tcp.lock.Unlock()
		case <-to.C://time.After(time.Second*3):
		}
	}
}

// 打包tcp响应包 格式为 [包长度-2字节，大端序][指令-2字节][内容]
func (tcp *TcpService) pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m)
	r := make([]byte, l + 6)

	cl := l + 2

	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 32)

	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], m)

	return r
}

func (tcp *TcpService) pack2(cmd int, msg []byte, table []byte) []byte {
	l  := len(msg)
	tl := len(table)
	r  := make([]byte, l + 6 + tl)

	cl := l + 2

	copy(r[0:], table)

	r[tl+0] = byte(cl)
	r[tl+1] = byte(cl >> 8)
	r[tl+2] = byte(cl >> 16)
	r[tl+3] = byte(cl >> 32)

	r[tl+4] = byte(cmd)
	r[tl+5] = byte(cmd >> 8)
	copy(r[tl+6:], msg)

	return r
}


// 掉线回调
func (tcp *TcpService) onClose(conn *tcp_client_node) {
	if conn.group == "" {
		tcp.lock.Lock()
		conn.is_connected = false
		close(conn.send_queue)
		tcp.lock.Unlock()
		return
	}
	//移除conn
	//查实查找位置
	tcp.lock.Lock()
	close(conn.send_queue)
	for index, con := range tcp.groups[conn.group] {
		if con.conn == conn.conn {
			con.is_connected = false
			tcp.groups[conn.group] = append(tcp.groups[conn.group][:index], tcp.groups[conn.group][index+1:]...)
			break
		}
	}
	tcp.lock.Unlock()
	atomic.AddInt32(&tcp.clients_count, int32(-1))
	log.Println("当前连输的客户端：", len(tcp.groups[conn.group]), tcp.groups[conn.group])
}

// 客户端服务协程，一个客户端一个
func (tcp *TcpService) clientSendService(node *tcp_client_node) {
	to := time.NewTimer(time.Second*1)
	for {
		if !node.is_connected {
			log.Println("clientSendService退出")
			return
		}

		select {
			case  msg, ok := <-node.send_queue:
				if !ok {
					log.Println("tcp发送消息channel通道关闭")
					return
				}
				(*node.conn).SetWriteDeadline(time.Now().Add(time.Second*1))
				size, err := (*node.conn).Write(msg)
				atomic.AddInt64(&node.send_times, int64(1))
				
				if (size <= 0 || err != nil) {
					atomic.AddInt64(&tcp.send_failure_times, int64(1))
					atomic.AddInt64(&node.send_failure_times, int64(1))
					
					log.Println((*node.conn).RemoteAddr().String(), "失败次数：", node.send_failure_times)
				}
			case <-to.C://time.After(time.Second*3):
				//log.Println("发送超时...", tcp)
		}
	}
}

// 连接成功回调
func (tcp *TcpService) onConnect(conn net.Conn) {
	log.Println("新的连接：",conn.RemoteAddr().String())

	cnode := &tcp_client_node {
		conn               : &conn,
		is_connected       : true,
		send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
		send_failure_times : 0,
		connect_time       : time.Now().Unix(),
		send_times         : int64(0),
		recv_buf           : make([]byte, TCP_RECV_DEFAULT_SIZE),
		recv_bytes         : 0,
		group              : "",
	}


	go tcp.clientSendService(cnode)
	var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte

	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	conn.SetReadDeadline(time.Now().Add(time.Second*3))
	for {
		buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据 memset
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
		log.Println("收到消息",size,"字节：", buf[:size], string(buf))
		atomic.AddInt64(&tcp.recv_times, int64(1))
		cnode.recv_bytes += size
		tcp.onMessage(cnode, buf, size)
	}
}

// 收到消息回调函数
func (tcp *TcpService) onMessage(conn *tcp_client_node, msg []byte, size int) {
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
				log.Println("收到注册分组消息")
				if len(conn.recv_buf) < 6 {
					return
				}
				
				//内容长度+4字节的前缀（存放内容长度的数值）
				//group := string(conn.recv_buf[6:content_len + 4])
				group := string(conn.recv_buf[6:content_len + 4])
				log.Println("group：", group)
				
				tcp.lock.Lock()
				is_find := false
				for g, _ := range tcp.groups {
					//log.Println(g, len(g), ">" + g + "<", len(group), ">" + group + "<")
					if g == group {
						is_find = true
						break
					}
				}
				if !is_find {
					conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("组不存在：%s", group))
					tcp.lock.Unlock()
					return
				}
				
				(*conn.conn).SetReadDeadline(time.Time{})
				conn.send_queue <- tcp.pack(CMD_SET_PRO, "ok")
				
				conn.group = group
				
				tcp.groups[group] = append(tcp.groups[group], conn)
				
				atomic.AddInt32(&tcp.clients_count, int32(1))
				tcp.lock.Unlock()
			case CMD_TICK:
				log.Println("收到心跳消息")
				conn.send_queue <- tcp.pack(CMD_TICK, "ok")
			//心跳包
			default:
				conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
		}

		//数据移动
		//log.Println(content_len + 4, conn.recv_bytes)
		conn.recv_buf = append(conn.recv_buf[:0], conn.recv_buf[content_len + 4:conn.recv_bytes]...)
		conn.recv_bytes = conn.recv_bytes - content_len - 4
		//log.Println("移动后的数据：", conn.recv_bytes, len(conn.recv_buf), string(conn.recv_buf))
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
