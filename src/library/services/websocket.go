package services

import (
	"context"
	"fmt"
	"github.com/go-martini/martini"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"regexp"
	"sync"
	"sync/atomic"
	"time"
)

func NewWebSocketService(ctx *context.Context) *WebSocketService {
	config, _ := getWebsocketConfig()
	tcp := &WebSocketService{
		Ip:               config.Listen,
		Port:             config.Port,
		clientsCount:     int32(0),
		lock:             new(sync.Mutex),
		groups:           make(map[string][]*websocketClientNode),
		groupsMode:       make(map[string]int),
		groupsFilter:     make(map[string][]string),
		recvTimes:        0,
		sendTimes:        0,
		sendFailureTimes: 0,
		enable:           config.Enable,
		wg:               new(sync.WaitGroup),
		ctx:              ctx,
	}
	for _, v := range config.Groups {
		var con [TCP_DEFAULT_CLIENT_SIZE]*websocketClientNode
		tcp.groups[v.Name] = con[:0]
		tcp.groupsMode[v.Name] = v.Mode
		flen := len(v.Filter)
		tcp.groupsFilter[v.Name] = make([]string, flen)
		tcp.groupsFilter[v.Name] = append(tcp.groupsFilter[v.Name][:0], v.Filter...)
	}
	return tcp
}

// 对外的广播发送接口
func (tcp *WebSocketService) SendAll(msg []byte) bool {
	if !tcp.enable {
		return false
	}
	log.Debugf("websocket服务-发送广播：", string(msg))
	cc := atomic.LoadInt32(&tcp.clientsCount)
	if cc <= 0 {
		log.Debugf("websocket服务-没有连接的客户端")
		return false
	}
	//if len(tcp.send_queue) >= cap(tcp.send_queue) {
	//	log.Debugf("websocket服务-发送缓冲区满...")
	//	return false
	//}
	table_len := int(msg[0]) + int(msg[1]<<8)
	table := string(msg[2 : table_len+2])

	//tcp.send_queue <- tcp.pack2(CMD_EVENT, msg[table_len+2:], msg[:table_len+2])

	tcp.lock.Lock()
	for group_name, cgroup := range tcp.groups {
		// 如果分组里面没有客户端连接，跳过
		if len(cgroup) <= 0 {
			continue
		}
		// 分组的模式
		mode := tcp.groupsMode[group_name]
		filter := tcp.groupsFilter[group_name]
		flen := len(filter)
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
			for _, cnode := range cgroup {
				if !cnode.isConnected {
					continue
				}
				log.Debugf("websocket服务-发送广播消息")
				if len(cnode.sendQueue) >= cap(cnode.sendQueue) {
					log.Warnf("websocket服务-发送缓冲区满：%s", (*cnode.conn).RemoteAddr().String())
					continue
				}
				cnode.sendQueue <- tcp.pack(CMD_EVENT, string(msg[table_len+2:])) //msg[table_len+2:]
			}
		} else {
			// 负载均衡模式
			// todo 根据已经send_times的次数负载均衡
			cc := len(cgroup)
			target := cgroup[0]
			//将发送次数/权重 作为负载基数，每次选择最小的发送
			js := float64(atomic.LoadInt64(&target.sendTimes)) / float64(target.weight)

			for i := 1; i < cc; i++ {
				stimes := atomic.LoadInt64(&cgroup[i].sendTimes)
				if stimes == 0 {
					//优先发送没有发过的
					target = cgroup[i]
					break
				}
				njs := float64(stimes) / float64(cgroup[i].weight)
				if njs < js {
					js = njs
					target = cgroup[i]
				}
			}
			log.Debugf("websocket服务-发送权重消息，%s", (*target.conn).RemoteAddr().String())
			if len(target.sendQueue) >= cap(target.sendQueue) {
				log.Warnf("websocket服务-发送缓冲区满：%s", (*target.conn).RemoteAddr().String())
			} else {
				target.sendQueue <- tcp.pack(CMD_EVENT, string(msg[table_len+2:]))
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
	r := make([]byte, l+2)
	r[0] = byte(cmd)
	r[1] = byte(cmd >> 8)
	copy(r[2:], m)
	return r
}

func (tcp *WebSocketService) onClose(node *websocketClientNode) {
	if node.group == "" {
		tcp.lock.Lock()
		node.isConnected = false
		close(node.sendQueue)
		tcp.lock.Unlock()
		return
	}
	//移除conn
	//查实查找位置
	tcp.lock.Lock()
	close(node.sendQueue)
	for index, cnode := range tcp.groups[node.group] {
		if cnode.conn == node.conn {
			cnode.isConnected = false
			tcp.groups[node.group] = append(tcp.groups[node.group][:index], tcp.groups[node.group][index+1:]...)
			break
		}
	}
	tcp.lock.Unlock()
	atomic.AddInt32(&tcp.clientsCount, int32(-1))
	log.Println("websocket服务-当前连输的客户端：", len(tcp.groups[node.group]), tcp.groups[node.group])
}

func (tcp *WebSocketService) clientSendService(node *websocketClientNode) {
	tcp.wg.Add(1)
	defer tcp.wg.Done()
	for {
		if !node.isConnected {
			log.Warnf("websocket服务-clientSendService退出")
			return
		}

		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Warnf("websocket服务-发送消息channel通道关闭")
				return
			}
			(*node.conn).SetWriteDeadline(time.Now().Add(time.Second * 1))
			err := (*node.conn).WriteMessage(1, msg)
			atomic.AddInt64(&node.sendTimes, int64(1))
			if err != nil {
				atomic.AddInt64(&tcp.sendFailureTimes, int64(1))
				atomic.AddInt64(&node.sendFailureTimes, int64(1))
				log.Println("websocket服务-发送失败次数：", tcp.sendFailureTimes, node.conn.RemoteAddr().String(), node.sendFailureTimes)
			}
		case <-(*tcp.ctx).Done():
			if len(node.sendQueue) <= 0 {
				log.Warnf("websocket服务-clientSendService退出")
				return
			}

		}

	}
}

func (tcp *WebSocketService) onConnect(conn *websocket.Conn) {
	log.Infof("websocket服务-新的连接：", conn.RemoteAddr().String())
	cnode := &websocketClientNode{
		conn:             conn,
		isConnected:      true,
		sendQueue:        make(chan []byte, TCP_MAX_SEND_QUEUE),
		sendFailureTimes: 0,
		weight:           0,
		mode:             MODEL_BROADCAST,
		connectTime:      time.Now().Unix(),
		sendTimes:        int64(0),
		recvBytes:        0,
		group:            "",
	}
	go tcp.clientSendService(cnode)
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	conn.SetReadDeadline(time.Now().Add(time.Second * 3))
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
		atomic.AddInt64(&tcp.recvTimes, int64(1))
		cnode.recvBytes += size
		tcp.onMessage(cnode, message, size)
		select {
		case <-(*tcp.ctx).Done():
			log.Debugf("websocket服务-onConnect退出")
			return
		default:
		}
	}
}

// 收到消息回调函数
func (tcp *WebSocketService) onMessage(node *websocketClientNode, msg []byte, size int) {
	clen := len(msg)
	if clen < 2 {
		return
	}
	//2字节 command
	cmd := int(msg[0]) + int(msg[1]<<8)
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
			int(msg[3]<<8) +
			int(msg[4]<<16) +
			int(msg[5]<<32)
		if weight < 0 || weight > 100 {
			node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("websocket服务-不支持的权重值：%d，请设置为0-100之间", weight))
			return
		}
		//内容长度+4字节的前缀（存放内容长度的数值）
		group := string(msg[6:])
		log.Debugf("websocket服务-group：%s", group)
		tcp.lock.Lock()
		if _, ok := tcp.groups[group]; !ok {
			node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("组不存在：%s", group))
			tcp.lock.Unlock()
			return
		}
		(*node.conn).SetReadDeadline(time.Time{})
		node.sendQueue <- tcp.pack(CMD_SET_PRO, "ok")
		node.group = group
		node.mode = tcp.groupsMode[group]
		node.weight = weight
		tcp.groups[group] = append(tcp.groups[group], node)
		if node.mode == MODEL_WEIGHT {
			//weight 合理性格式化，保证所有的weight的和是100
			all_weight := 0
			for _, cnode := range tcp.groups[group] {
				w := cnode.weight
				if w <= 0 {
					w = 100
				}
				all_weight += w
			}
			gl := len(tcp.groups[group])
			yg := 0
			for k, cnode := range tcp.groups[group] {
				if k == gl-1 {
					cnode.weight = 100 - yg
				} else {
					cnode.weight = int(cnode.weight * 100 / all_weight)
					yg += cnode.weight
				}
			}
		}
		atomic.AddInt32(&tcp.clientsCount, int32(1))
		tcp.lock.Unlock()
	case CMD_TICK:
		//心跳包
		node.sendQueue <- tcp.pack(CMD_TICK, "ok")
	default:
		node.sendQueue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
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
			select {
			case <-(*tcp.ctx).Done():
				return
			default:
			}
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
	}()
}

func (tcp *WebSocketService) Close() {
	log.Debug("websocket服务退出...")
	log.Debug("websocket服务退出，等待缓冲区发送完毕")
	cc := atomic.LoadInt32(&tcp.clientsCount)
	if cc > 0 {
		tcp.wg.Wait()
	}
	tcp.lock.Lock()
	for _, cgroup := range tcp.groups {
		for _, cnode := range cgroup {
			log.Debugf("websocket关闭连接：%s", (*cnode.conn).RemoteAddr().String())
			(*cnode.conn).Close()
			cnode.isConnected = false
		}
	}
	tcp.lock.Unlock()
	log.Debug("websocket服务退出...end")
}

func (tcp *WebSocketService) Reload() {
	config, _ := getWebsocketConfig()
	log.Debug("websocket服务reload...")
	tcp.enable = config.Enable
	restart := false
	if config.Listen != tcp.Ip || config.Port != tcp.Port {
		// 需要重启
		restart = true
		tcp.clientsCount = 0
		tcp.recvTimes = 0
		tcp.sendTimes = 0
		tcp.sendFailureTimes = 0
		//如果端口和监听ip发生变化，则需要清理所有的客户端连接
		for k, v := range tcp.groups {
			for _, conn := range v {
				log.Debugf("websocket服务关闭连接：%s", (*conn.conn).RemoteAddr().String())
				(*conn.conn).Close()
			}
			log.Debugf("websocket服务删除分组：%s", k)
			delete(tcp.groups, k)
		}
		for k := range tcp.groupsMode {
			log.Debugf("websocket服务删除分组模式：%s", k)
			delete(tcp.groupsMode, k)
		}
		for k := range tcp.groupsFilter {
			log.Debugf("websocket服务删除分组过滤器：%s", k)
			delete(tcp.groupsFilter, k)
		}
		for _, v := range config.Groups {
			var con [TCP_DEFAULT_CLIENT_SIZE]*websocketClientNode
			tcp.groups[v.Name] = con[:0]
			tcp.groupsMode[v.Name] = v.Mode
			flen := len(v.Filter)
			tcp.groupsFilter[v.Name] = make([]string, flen)
			tcp.groupsFilter[v.Name] = append(tcp.groupsFilter[v.Name][:0], v.Filter...)
		}
	} else {
		//如果监听ip和端口没发生变化，则需要diff分组
		//检测是否发生删除和新增分组
		for _, v := range config.Groups {
			in_array := false
			for k := range tcp.groups {
				if k == v.Name {
					in_array = true
					break
				}
			}
			// 新增的分组
			if !in_array {
				log.Debugf("websocket服务新增分组：%s", v.Name)
				flen := len(v.Filter)
				var con [TCP_DEFAULT_CLIENT_SIZE]*websocketClientNode
				tcp.groups[v.Name] = con[:0]
				tcp.groupsMode[v.Name] = v.Mode
				tcp.groupsFilter[v.Name] = make([]string, flen)
				tcp.groupsFilter[v.Name] = append(tcp.groupsFilter[v.Name][:0], v.Filter...)
			}
		}
		for k, conns := range tcp.groups {
			in_array := false
			for _, v := range config.Groups {
				if k == v.Name {
					in_array = true
					break
				}
			}
			//被删除的分组
			if !in_array {
				log.Debugf("websocket服务移除的分组：%s", k)
				for _, conn := range conns {
					log.Debugf("websocket服务关闭连接：%s", (*conn.conn).RemoteAddr().String())
					(*conn.conn).Close()
				}
				delete(tcp.groups, k)
			}
		}
	}

	if restart {
		tcp.Close()
		tcp.Start()
	}
}
