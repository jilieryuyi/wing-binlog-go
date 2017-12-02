package http
// 仅供admin管理交互使用

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
	"time"
	"sync/atomic"
	ssh "library/wssh"
)

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
		clients            : make(map[string] *websocketClientNode, 128),
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
		log.Warnf("websocket服务发送缓冲区满...")
		return false
	}
	tcp.send_queue <- tcp.pack(CMD_EVENT, string(msg))
	return true
}

func (tcp *WebSocketService) broadcast() {
	for {
		select {
		case  msg := <-tcp.send_queue:
			tcp.lock.Lock()
			for _, conn := range tcp.clients {
				if !conn.is_connected {
					continue
				}
				log.Debugf("websocket服务发送广播消息")
				conn.send_queue <- msg
			}
			tcp.lock.Unlock()
		}
	}
}

// 数据打包
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
	//移除conn
	//查实查找位置
	tcp.lock.Lock()
	close(conn.send_queue)
	for index, con := range tcp.clients {
		if con == conn {
			con.is_connected = false
			delete(tcp.clients, index)
			log.Warnf("websocket服务连接发生错误，删除资源 %d", index)
			break
		}
	}
	tcp.lock.Unlock()
	atomic.AddInt32(&tcp.clients_count, int32(-1))
	log.Debugf("websocket服务当前连输的客户端：%d %v", len(tcp.clients), tcp.clients)
}

func (tcp *WebSocketService) clientSendService(node *websocketClientNode) {
	for {
		if !node.is_connected {
			log.Debugf("websocket服务clientSendService退出")
			return
		}
		select {
		case  msg, ok:= <-node.send_queue:
			if !ok {
				log.Debugf("websocket服务通道关闭")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second*1))
			err := (*node.conn).WriteMessage(1, msg)
			atomic.AddInt64(&node.send_times, int64(1))
			if (err != nil) {
				atomic.AddInt64(&tcp.send_failure_times, int64(1))
				atomic.AddInt64(&node.send_failure_times, int64(1))
				log.Warnf("websocket服务发送失败：%v", msg)
			}
		}
	}
}

func (tcp *WebSocketService) onConnect(conn *websocket.Conn) {
	log.Infof("websocket服务新的连接：%s", conn.RemoteAddr().String())
	cnode := &websocketClientNode {
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
			log.Errorf("%s websocket服务连接发生错误: %v", conn.RemoteAddr().String(), err)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Errorf("websocket服务error: %v", err)
			}
			tcp.onClose(cnode)
			conn.Close();
			return
		}
		size := len(message)
		atomic.AddInt64(&tcp.recv_times, int64(1))
		cnode.recv_bytes += size
		tcp.onMessage(cnode, message, size)
	}
}

func (tcp *WebSocketService) DeleteClient(key string) {
	log.Infof("websocket服务删除ws资源 %s", key)
	tcp.lock.Lock()
	conn, ok := tcp.clients[key]
	if ok {
		//close(conn.send_queue) 这里不需要close，因为接下来的
		//conn.Close()会触发onClose事件，这个事件里面会关闭channel
		conn.conn.Close()
	}
	tcp.lock.Unlock()
}

// 收到消息回调函数
func (tcp *WebSocketService) onMessage(conn *websocketClientNode, msg []byte, size int) {
	log.Debugf("websocket服务收到消息 %d %v %s", len(msg), msg, string(msg))
	clen := len(msg)
	if clen < 34 {
		return
	}
	cmd := int(msg[0]) + int(msg[1] << 8) //2字节 command
	sign := string(msg[2:34])             //签名
	log.Debugf("websocket服务签名：%s", sign)
	if !isOnline(sign) {
		conn.send_queue <- tcp.pack(CMD_RELOGIN, "需要重新登录")
		return
	}
	switch cmd {
	case CMD_AUTH:
		tcp.lock.Lock()
		(*conn.conn).SetReadDeadline(time.Time{})
		conn.send_queue <- tcp.pack(CMD_AUTH, "ok")
		tcp.clients[sign] = conn
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
		conn.send_queue <- tcp.pack(CMD_TICK, "ok")
	default:
		conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
	}
}

func (tcp *WebSocketService) Start() {
	log.Debugf("websocket服务等待新的连接...")
	go tcp.broadcast()
	go func() {
		m := martini.Classic()
		m.Get("/", func(res http.ResponseWriter, req *http.Request) {
			u := websocket.Upgrader{ReadBufferSize: TCP_DEFAULT_READ_BUFFER_SIZE,
				WriteBufferSize: TCP_DEFAULT_WRITE_BUFFER_SIZE}
			u.Error = func(w http.ResponseWriter, r *http.Request, status int, reason error) {
				log.Println("websocket服务", w, r, status, reason)
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
			log.Infof("websocket服务新的连接：%s", conn.RemoteAddr().String())
			go tcp.onConnect(conn)
		})
		dns := fmt.Sprintf("%s:%d", tcp.Ip, tcp.Port)
		log.Infof("websocket服务监听: %s", dns)
		m.RunOnAddr(dns)
	} ()
}
