package services

import (
	"fmt"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
	"runtime"
	"sync/atomic"
)

type websocket_client_node struct {
	conn *websocket.Conn
	is_connected bool
	send_queue chan []byte
	send_failure_times int64
}

const (
	WEBSOCKET_MAX_SEND_QUEUE = 4096
	WEBSOCKET_DEFAULT_CLIENT_SIZE = 64
	WEBSOCKET_DEFAULT_READ_BUFFER_SIZE  = 4096
	WEBSOCKET_DEFAULT_WRITE_BUFFER_SIZE = 4096
)


type WebSocketService struct {
	Ip string
	Port int
	clients []*websocket_client_node
	recv_times int64
	send_times int64
	send_failure_times int64
	send_queue chan []byte
	lock *sync.Mutex
}

func NewWebSocketService(ip string, port int) *WebSocketService {
	tcp := &WebSocketService{
		Ip:ip,
		Port:port,
	}

	var con [WEBSOCKET_DEFAULT_CLIENT_SIZE]*websocket_client_node
	tcp.clients = con[:0]
	tcp.recv_times = 0
	tcp.send_times = 0
	tcp.send_failure_times = 0
	tcp.send_queue = make(chan []byte, WEBSOCKET_MAX_SEND_QUEUE)
	tcp.lock = new(sync.Mutex)
	return tcp
}

/**
 * 对外的广播发送接口
 */
func (ws *WebSocketService) SendAll(msg []byte) bool {
	clen := len(ws.clients)
	if clen <=0 {
		return false
	}
	if len(ws.send_queue) >= cap(ws.send_queue) {
		log.Println("websocket发送缓冲区满...", clen)
		return false
	}
	ws.send_queue <- msg
	return true
}

func (tcp *WebSocketService) clientSendService(node *websocket_client_node) {
	to := time.NewTimer(time.Second*1)

	for {
		if !node.is_connected {
			break
		}

		select {
		case  msg := <-node.send_queue:
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second*1))
			err := (*node.conn).WriteMessage(1, msg)
			if err != nil {
				atomic.AddInt64(&tcp.send_failure_times, int64(1))
				atomic.AddInt64(&node.send_failure_times, int64(1))

				log.Println("失败次数：", tcp.send_failure_times, node.conn, node.send_failure_times)
			}
		case <-to.C://time.After(time.Second*3):
		//log.Println("发送超时...", tcp)
		}
	}
}

func (ws *WebSocketService) onConnect(conn *websocket.Conn) {

	//clients[clients_count] = conn
	//clients_count++
	//var buffer bytes.Buffer
	//body := BODY{conn, buffer}
	ws.lock.Lock()
	cnode := &websocket_client_node{conn, true, make(chan []byte, WEBSOCKET_MAX_SEND_QUEUE), 0}
	ws.clients = append(ws.clients, cnode)
	ws.lock.Unlock()


	go ws.clientSendService(cnode)

	//var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			ws.onClose(conn)
			conn.Close();
			break
		}
		log.Println("收到websocket消息：", string(message))
		ws.onMessage(message)
	}
}

func (ws *WebSocketService) onClose(conn *websocket.Conn) {
	//移除conn
	//查实查找位置
	ws.lock.Lock()
	for index, con := range ws.clients {
		if con.conn == conn {
			con.is_connected = false
			ws.clients = append(ws.clients[:index], ws.clients[index+1:]...)
			break
		}
	}
	ws.lock.Unlock()
}

func (ws *WebSocketService) onMessage(msg []byte) {

}

/**
 * 广播服务
 */
func (ws *WebSocketService) broadcast() {
	to := time.NewTimer(time.Second*1)
	cpu := runtime.NumCPU()
	for i := 0; i < cpu; i ++ {
		tou := to
		go func() {
			for {
				select {
				case  msg := <-ws.send_queue:
					for _, conn := range ws.clients {
						conn.send_queue <- msg
					}
				case <-tou.C://time.After(time.Second*3):
				}
			}
		} ()
	}
}

func (ws *WebSocketService) Start() {

	go ws.broadcast()

	m := martini.Classic()

	m.Get("/", func(res http.ResponseWriter, req *http.Request) {
		// res and req are injected by Martini

		u := websocket.Upgrader{ReadBufferSize: WEBSOCKET_DEFAULT_READ_BUFFER_SIZE,
			WriteBufferSize: WEBSOCKET_DEFAULT_WRITE_BUFFER_SIZE}
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
		go ws.onConnect(conn)
	})

	dns := fmt.Sprintf("%s:%d", ws.Ip, ws.Port)
	m.RunOnAddr(dns)
}