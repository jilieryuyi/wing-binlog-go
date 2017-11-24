package http
// 仅供admin管理交互使用

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
	ssh "library/wssh"
)

type websocket_client_node struct {
	conn *websocket.Conn     // 客户端连接进来的资源句柄
	is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
	send_queue chan []byte   // 发送channel
	send_failure_times int64 // 发送失败次数
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
	clients_count int32                   // 成功连接（已经进入分组）的客户端数量
	clients map[string] *websocket_client_node
	//http *HttpServer
}

func NewWebSocketService(ip string, port int) *WebSocketService {
	tcp := &WebSocketService {
		Ip                 : ip,
		Port               : port,
		clients_count      : int32(0),
		lock               : new(sync.Mutex),
		send_queue         : make(chan []byte, 128),
		recv_times         : 0,
		send_times         : 0,
		send_failure_times : 0,
		clients            : make(map[string] *websocket_client_node, 128),
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
		log.Println("websocket(http服务)发送缓冲区满...")
		return false
	}

	tcp.send_queue <- tcp.pack(CMD_EVENT, string(msg))
	return true
}

func (tcp *WebSocketService) broadcast() {
	to := time.NewTimer(time.Second*1)
	for {
		select {
		case  msg := <-tcp.send_queue:
			tcp.lock.Lock()
			for _, conn := range tcp.clients {
				if !conn.is_connected {
					continue
				}
				log.Println("发送广播消息")
				conn.send_queue <- msg
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

func (tcp *WebSocketService) onClose(conn *websocket_client_node) {
	//移除conn
	//查实查找位置
	tcp.lock.Lock()
	close(conn.send_queue)
	for index, con := range tcp.clients {
		if con == conn {
			con.is_connected = false
			delete(tcp.clients, index)
			log.Println("连接发生错误，删除资源", index)
			break
		}
	}
	tcp.lock.Unlock()
	atomic.AddInt32(&tcp.clients_count, int32(-1))
	log.Println("当前连输的客户端：", len(tcp.clients), tcp.clients)
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
				log.Println("通道关闭")
				// 通道已关闭
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second*1))
			//size, err := (*node.conn).Write(msg)
			err := (*node.conn).WriteMessage(1, msg)

			atomic.AddInt64(&node.send_times, int64(1))

			if (err != nil) {
				atomic.AddInt64(&tcp.send_failure_times, int64(1))
				atomic.AddInt64(&node.send_failure_times, int64(1))
				log.Println("websocket-发送失败：", msg)
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
		connect_time       : time.Now().Unix(),
		send_times         : int64(0),
		recv_bytes         : 0,
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
		//log.Println("收到websocket消息：", string(message))

		size := len(message)
		atomic.AddInt64(&tcp.recv_times, int64(1))
		cnode.recv_bytes += size
		tcp.onMessage(cnode, message, size)
	}
}
func (tcp *WebSocketService) DeleteClient(key string) {
	log.Println("删除ws资源", key)
	tcp.lock.Lock()
	conn, ok := tcp.clients[key]
	if ok {
		//close(conn.send_queue) 这里不需要close，因为接下来的
		//conn.Close()会触发onClose事件，这个事件里面会关闭channel
		conn.conn.Close()
		//conn.is_connected = false
		//delete(tcp.clients, key)
	}
	tcp.lock.Unlock()
}

// 收到消息回调函数
func (tcp *WebSocketService) onMessage(conn *websocket_client_node, msg []byte, size int) {
	log.Println("收到消息",len(msg), "==>",msg)
	clen := len(msg)
	if clen < 34 { // 2字节cmd，32字节签名
		return
	}

	//2字节 command
	cmd := int(msg[0]) + int(msg[1] << 8)
	//签名
	sign := string(msg[2:34])
	log.Println("签名：",sign)
	if !is_online(sign) {
		conn.send_queue <- tcp.pack(CMD_RELOGIN, "需要重新登录")
		return
	}

	//log.Println("content：", msg)
	//log.Println("cmd：", cmd)
	switch cmd {
	case CMD_AUTH:
		//log.Println("收到认证消息")
		//if len(msg) < 3 {
		//	return
		//}

		//签名
		//sign := string(msg[2:])
		//log.Println("sign：", sign)
		tcp.lock.Lock()
		(*conn.conn).SetReadDeadline(time.Time{})
		conn.send_queue <- tcp.pack(CMD_AUTH, "ok")
		//log.Println("发送CMD_AUTH ok")
		tcp.clients[sign] = conn// = append(tcp.clients, conn)
		atomic.AddInt32(&tcp.clients_count, int32(1))
		tcp.lock.Unlock()
	case CMD_CONNECT:
		client := &ssh.SSH{
			Ip: "127.0.0.1",
			User : "root",
			Port:22,
			Cert:"123456",
		}
		client.Connect(ssh.CERT_PASSWORD)
		client.RunCmd("ls /home")
		client.Close()
	case CMD_TICK:
		//log.Println("收到心跳消息")
		conn.send_queue <- tcp.pack(CMD_TICK, "ok")
		//log.Println("发送CMD_TICK ok")
	//心跳包
	default:
		conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
		//log.Println("发送CMD_ERROR ",fmt.Sprintf("不支持的指令：%d", cmd))
	}
}

func (tcp *WebSocketService) Start() {
	log.Println("admin websocket等待新的连接...")

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
		log.Println("admin websocket listen: ", dns)
		m.RunOnAddr(dns)
	} ()
}