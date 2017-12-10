package services

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
	"sync/atomic"
	"regexp"
	"context"
)

func NewWebSocketService() *WebSocketService {
	config, _ := getWebsocketConfig()
	tcp := &WebSocketService {
		Ip                 : config.Tcp.Listen,
		Port               : config.Tcp.Port,
		clients_count      : int32(0),
		lock               : new(sync.Mutex),
		groups             : make(map[string][]*websocketClientNode),
		groups_mode        : make(map[string] int),
		groups_filter      : make(map[string] []string),
		recv_times         : 0,
		send_times         : 0,
		send_failure_times : 0,
		enable             : config.Enable,
	}
	for _, v := range config.Groups {
		var con [TCP_DEFAULT_CLIENT_SIZE]*websocketClientNode
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
	if !tcp.enable {
		return false
	}
	log.Debugf("websocket服务-发送广播：", string(msg))
	cc := atomic.LoadInt32(&tcp.clients_count)
	if cc <= 0 {
		log.Debugf("websocket服务-没有连接的客户端")
		return false
	}
	//if len(tcp.send_queue) >= cap(tcp.send_queue) {
	//	log.Debugf("websocket服务-发送缓冲区满...")
	//	return false
	//}
	table_len := int(msg[0]) + int(msg[1] << 8)
	table     := string(msg[2:table_len+2])

	//tcp.send_queue <- tcp.pack2(CMD_EVENT, msg[table_len+2:], msg[:table_len+2])

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
		log.Debugf("websocket服务-数据表：%s", table)
		log.Debugf("websocket服务：%v", filter)
		if flen > 0 {
			is_match := false
			for _, f := range filter {
				match, err := regexp.MatchString(f, table)
				if err != nil {
					continue
				}
				if match {
					is_match = true
					break
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
				log.Debugf("websocket服务-发送广播消息")
				if len(conn.send_queue) >= cap(conn.send_queue) {
					log.Warnf("websocket服务-发送缓冲区满：%s", (*conn.conn).RemoteAddr().String())
					continue
				}
				conn.send_queue <- tcp.pack(CMD_EVENT, string(msg[table_len+2:]))//msg[table_len+2:]
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
				if stimes == 0 {
					//优先发送没有发过的
					target = clients[i]
					break
				}
				njs := float64(stimes)/float64(clients[i].weight)
				if njs < js {
					js = njs
					target = clients[i]
				}
			}
			log.Debugf("websocket服务-发送权重消息，%s", (*target.conn).RemoteAddr().String())
			if len(target.send_queue) >= cap(target.send_queue) {
				log.Warnf("websocket服务-发送缓冲区满：%s", (*target.conn).RemoteAddr().String())
			} else {
				target.send_queue <- tcp.pack(CMD_EVENT, string(msg[table_len+2:]))
			}
		}
	}
	tcp.lock.Unlock()

	return true
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

func (tcp *WebSocketService) onClose(conn *websocketClientNode) {
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
	log.Println("websocket服务-当前连输的客户端：", len(tcp.groups[conn.group]), tcp.groups[conn.group])
}

func (tcp *WebSocketService) clientSendService(node *websocketClientNode) {
	for {
		if !node.is_connected {
			log.Warnf("websocket服务-clientSendService退出")
			return
		}
		select {
		case  msg, ok:= <-node.send_queue:
			if !ok {
				log.Warnf("websocket服务-发送消息channel通道关闭")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second*1))
			err := (*node.conn).WriteMessage(1, msg)
			atomic.AddInt64(&node.send_times, int64(1))
			if err != nil {
				atomic.AddInt64(&tcp.send_failure_times, int64(1))
				atomic.AddInt64(&node.send_failure_times, int64(1))
				log.Println("websocket服务-发送失败次数：", tcp.send_failure_times, node.conn.RemoteAddr().String(), node.send_failure_times)
			}
		}
	}
}

func (tcp *WebSocketService) onConnect(conn *websocket.Conn) {
	log.Infof("websocket服务-新的连接：", conn.RemoteAddr().String())
	cnode := &websocketClientNode {
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
			log.Println(conn.RemoteAddr().String(), "websocket服务-连接发生错误: ", err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Errorf("websocket服务-error: %v", err)
			}
			tcp.onClose(cnode)
			conn.Close()
			return
		}
		log.Debugf("websocket服务-收到消息：%s", string(message))
		size := len(message)
		atomic.AddInt64(&tcp.recv_times, int64(1))
		cnode.recv_bytes += size
		tcp.onMessage(cnode, message, size)
	}
}

// 收到消息回调函数
func (tcp *WebSocketService) onMessage(conn *websocketClientNode, msg []byte, size int) {
	clen := len(msg)
	if clen < 2 {
		return
	}
	//2字节 command
	cmd := int(msg[0]) + int(msg[1] << 8)
	log.Debugf("websocket服务-content：%v", msg)
	log.Debugf("websocket服务-cmd：%d", cmd)
	switch cmd {
	case CMD_SET_PRO:
		log.Infof("websocket服务-收到注册分组消息")
		if len(msg) < 6 {
			return
		}
		//4字节 weight
		weight := int(msg[2]) +
			int(msg[3] << 8) +
			int(msg[4] << 16) +
			int(msg[5] << 32)
		if weight < 0 || weight > 100 {
			conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("websocket服务-不支持的权重值：%d，请设置为0-100之间", weight))
			return
		}
		//内容长度+4字节的前缀（存放内容长度的数值）
		group := string(msg[6:])
		log.Debugf("websocket服务-group：%s", group)
		tcp.lock.Lock()
		if _, ok := tcp.groups[group]; !ok {
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
		//心跳包
		conn.send_queue <- tcp.pack(CMD_TICK, "ok")
	default:
		conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
	}
}

func (tcp *WebSocketService) Start() {
	if !tcp.enable {
		return
	}
	log.Infof("websocket服务-等待新的连接...")
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
			log.Infof("websocket服务-新的连接：%s", conn.RemoteAddr().String())
			go tcp.onConnect(conn)
		})
		dns := fmt.Sprintf("%s:%d", tcp.Ip, tcp.Port)
		log.Infof("websocket服务-监听: %s", dns)
		m.RunOnAddr(dns)
	} ()
}

func (tcp *WebSocketService) Close() {
	log.Debug("websocket服务退出...")
}

func (tcp *WebSocketService) SetContext(ctx *context.Context) {
	tcp.ctx = ctx
}

func (tcp *WebSocketService) Reload() {

}