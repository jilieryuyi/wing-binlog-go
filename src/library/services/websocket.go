package services

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	//log "library/log"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
	"sync/atomic"
	"regexp"
)

type websocket_client_node struct {
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
	Ip string                             // 监听ip
	Port int                              // 监听端口
	recv_times int64                      // 收到消息的次数
	send_times int64                      // 发送消息的次数
	send_failure_times int64              // 发送失败的次数
	send_queue chan []byte                // 发送队列-广播
	lock *sync.Mutex                      // 互斥锁，修改资源时锁定
	groups map[string][]*websocket_client_node // 客户端分组，现在支持两种分组，广播组合负载均衡组
	groups_mode map[string] int           // 分组的模式 1，2 广播还是复载均衡
	groups_filter map[string] []string    // 分组的过滤器
	clients_count int32                   // 成功连接（已经进入分组）的客户端数量
}

func NewWebSocketService() *WebSocketService {
	//websocket_config.Tcp.Listen, websocket_config.Tcp.Port,
	config, _ := getWebsocketConfig()
	tcp := &WebSocketService {
		Ip                 : config.Tcp.Listen,
		Port               : config.Tcp.Port,
		clients_count      : int32(0),
		lock               : new(sync.Mutex),
		send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
		groups             : make(map[string][]*websocket_client_node),
		groups_mode        : make(map[string] int),
		groups_filter      : make(map[string] []string),
		recv_times         : 0,
		send_times         : 0,
		send_failure_times : 0,
	}

	for _, v := range config.Groups {
		var con [TCP_DEFAULT_CLIENT_SIZE]*websocket_client_node
		tcp.groups[v.Name]      = con[:0]
		tcp.groups_mode[v.Name] = v.Mode

		flen := len(v.Filter)
		tcp.groups_filter[v.Name] = make([]string, flen)
		tcp.groups_filter[v.Name] = append(tcp.groups_filter[v.Name][:0], v.Filter...)
	}

	return tcp
}

// 对外的广播发送接口
func (tcp *WebSocketService) SendAll(msg []byte) bool {
	cc := atomic.LoadInt32(&tcp.clients_count)
	if cc <= 0 {
		return false
	}
	if len(tcp.send_queue) >= cap(tcp.send_queue) {
		log.Println("websocket发送缓冲区满...")
		return false
	}

	table_len := int(msg[0]) + int(msg[1] << 8);
	tcp.send_queue <- tcp.pack2(CMD_EVENT, msg[table_len+2:], msg[:table_len+2])
	return true
}

func (tcp *WebSocketService) broadcast() {
	to := time.NewTimer(time.Second*1)
	for {
		select {
		case  msg := <-tcp.send_queue:
			tcp.lock.Lock()
			for group_name, clients := range tcp.groups {
				// 如果分组里面没有客户端连接，跳过
				if len(clients) <= 0 {
					continue
				}
				// 分组的模式
				mode   := tcp.groups_mode[group_name]
				filter := tcp.groups_filter[group_name]
				flen   := len(filter)

				//2字节长度
				table_len := int(msg[0]) + int(msg[1] << 8);
				table     := string(msg[2:table_len+2])

				log.Println("websocket数据表：", table)
				log.Println(filter)

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

				// 如果不等于权重，即广播模式
				if mode != MODEL_WEIGHT {
					for _, conn := range clients {
						if !conn.is_connected {
							continue
						}
						log.Println("发送广播消息")
						conn.send_queue <- msg[table_len+2:]
					}
				} else {
					// 负载均衡模式
					// todo 根据已经send_times的次数负载均衡
					clen := len(clients)
					target := clients[0]
					//将发送次数/权重 作为负载基数，每次选择最小的发送
					js := float64(atomic.LoadInt64(&target.send_times))/float64(target.weight)

					for i := 1; i < clen; i++ {
						stimes := atomic.LoadInt64(&clients[i].send_times)
						//conn.send_queue <- msg
						if stimes == 0 {
							//优先发送没有发过的
							target = clients[i]
							break
						}
						_js := float64(stimes)/float64(clients[i].weight)
						if _js < js {
							js = _js
							target = clients[i]
						}
					}
					log.Println("发送权重消息，", (*target.conn).RemoteAddr().String())
					target.send_queue <- msg[table_len+2:]
				}
			}
			tcp.lock.Unlock()
		case <-to.C://time.After(time.Second*3):
		}
	}
}

// 打包tcp响应包 格式为 [包长度-2字节，小端序][指令-2字节][内容]
func (tcp *WebSocketService) pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m)
	r := make([]byte, l + 2)

	r[0] = byte(cmd)
	r[1] = byte(cmd >> 8)
	copy(r[2:], m)

	return r
}

func (tcp *WebSocketService) pack2(cmd int, msg []byte, table []byte) []byte {
	l := len(msg)
	tl := len(table)
	r := make([]byte, l + 2 + tl)

	copy(r[0:], table)
	r[tl+0] = byte(cmd)
	r[tl+1] = byte(cmd >> 8)
	copy(r[tl+2:], msg)

	return r
}

func (tcp *WebSocketService) onClose(conn *websocket_client_node) {
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

func (tcp *WebSocketService) clientSendService(node *websocket_client_node) {
	to := time.NewTimer(time.Second*1)
	for {
		if !node.is_connected {
			log.Println("websocket-clientSendService退出")
			return
		}

		select {
		case  msg, ok:= <-node.send_queue:
			if !ok {
				log.Println("websocket发送消息channel通道关闭")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second*1))
			//size, err := (*node.conn).Write(msg)
			err := (*node.conn).WriteMessage(1, msg)

			atomic.AddInt64(&node.send_times, int64(1))

			if (err != nil) {
				atomic.AddInt64(&tcp.send_failure_times, int64(1))
				atomic.AddInt64(&node.send_failure_times, int64(1))

				log.Println("websocket-失败次数：", tcp.send_failure_times, node.conn.RemoteAddr().String(), node.send_failure_times)
			}
		case <-to.C://time.After(time.Second*3):
		//log.Println("发送超时...", tcp)
		}
	}
}

func (tcp *WebSocketService) onConnect(conn *websocket.Conn) {

	log.Println("新的连接：",conn.RemoteAddr().String())
	cnode := &websocket_client_node {
		conn               : conn,
		is_connected       : true,
		send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
		send_failure_times : 0,
		weight             : 0,
		mode               : MODEL_BROADCAST,
		connect_time       : time.Now().Unix(),
		send_times         : int64(0),
		recv_bytes         : 0,
		group              : "",
	}


	go tcp.clientSendService(cnode)
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	conn.SetReadDeadline(time.Now().Add(time.Second*3))
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(conn.RemoteAddr().String(), "连接发生错误: ", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			tcp.onClose(cnode)
			conn.Close();
			return
		}
		log.Println("收到websocket消息：", string(message))

		size := len(message)
		atomic.AddInt64(&tcp.recv_times, int64(1))
		cnode.recv_bytes += size
		tcp.onMessage(cnode, message, size)
	}
}

// 收到消息回调函数
func (tcp *WebSocketService) onMessage(conn *websocket_client_node, msg []byte, size int) {
	//for
	{
		clen := len(msg)
		if clen < 2 {
			return
		}

		//2字节 command
		cmd := int(msg[0]) + int(msg[1] << 8)

		log.Println("content：", msg)
		log.Println("cmd：", cmd)
		switch cmd {
		case CMD_SET_PRO:
			log.Println("收到注册分组消息")
			if len(msg) < 6 {
				return
			}
			//4字节 weight
			weight := int(msg[2]) +
				int(msg[3] << 8) +
				int(msg[4] << 16) +
				int(msg[5] << 32)

			//log.Println("weight：", weight)
			if weight < 0 || weight > 100 {
				conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的权重值：%d，请设置为0-100之间", weight))
				return
			}

			//内容长度+4字节的前缀（存放内容长度的数值）
			group := string(msg[6:])
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

			conn.group  = group
			conn.mode   = tcp.groups_mode[group]
			conn.weight = weight

			tcp.groups[group] = append(tcp.groups[group], conn)

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
			//log.Println("收到心跳消息")
			conn.send_queue <- tcp.pack(CMD_TICK, "ok")
		//心跳包
		default:
			conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
		}
	}
}

func (tcp *WebSocketService) Start() {
	log.Println("等待新的连接...")

	go tcp.broadcast()

	go func() {
		m := martini.Classic()

		m.Get("/", func(res http.ResponseWriter, req *http.Request) {
			// res and req are injected by Martini

			u := websocket.Upgrader{ReadBufferSize: TCP_DEFAULT_READ_BUFFER_SIZE,
				WriteBufferSize: TCP_DEFAULT_WRITE_BUFFER_SIZE}
			u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				log.Println(w, r, status, reason)
				// don't return errors to maintain backwards compatibility
			}
			u.CheckOrigin = func(r *http.Request) bool {
				// allow all connections by default
				return true
			}
			conn, err := u.Upgrade(res, req, nil)

			if err != nil {
				log.Println(err)
				return
			}

			log.Println("新的连接：" + conn.RemoteAddr().String())
			go tcp.onConnect(conn)
		})

		dns := fmt.Sprintf("%s:%d", tcp.Ip, tcp.Port)
		log.Println("websocket listen: ", tcp, dns)
		m.RunOnAddr(dns)
	} ()
}