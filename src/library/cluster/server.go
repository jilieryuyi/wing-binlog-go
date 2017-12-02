package cluster

import (
	"fmt"
	"net"
	"time"
	log "github.com/sirupsen/logrus"
	"library/buffer"
	"strconv"
)

func (server *tcp_server) start() {
	go func() {
		//建立socket，监听端口
		dns := fmt.Sprintf("%s:%d", server.listen, server.port)
		listen, err := net.Listen("tcp", dns)
		if err != nil {
			log.Println(err)
			return
		}
		defer listen.Close()
		log.Debugf("cluster server等待新的连接...")
		for {
			conn, err := listen.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			go server.onConnect(conn)
		}
	} ()
}

// 广播
func (server *tcp_server) send(cmd int, client_id []byte, msg []string){
	// todo 这里需要加入channel和select，超时机制
	server.lock.Lock()
	log.Println("---clients", server.clients)
	for _, client := range server.clients {
		send_msg := pack(cmd, string(client_id), msg)
		log.Debugf("cluster server发送消息：", len(send_msg), send_msg, string(send_msg))
		(*client.conn).Write(send_msg)
	}
	server.lock.Unlock()
}

func (server *tcp_server) onConnect(conn net.Conn) {
	log.Infof("cluster server新的连接：%s", conn.RemoteAddr().String())
	cnode := &tcp_client_node {
		conn               : &conn,
		is_connected       : true,
		send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
		send_failure_times : 0,
		weight             : 0,
		connect_time       : time.Now().Unix(),
		send_times         : int64(0),
		recv_buf           : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),//make([]byte, TCP_RECV_DEFAULT_SIZE),
	}
	server.lock.Lock()
	server.clients = append(server.clients[:server.clients_count], cnode)
	server.clients_count++
	server.lock.Unlock()
	var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
	// 设定3秒超时，如果添加到分组成功，超时限制将被清除
	//conn.SetReadDeadline(time.Now().Add(time.Second*3))
	for {
		buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
		//清空旧数据 memset
		for k,_:= range buf {
			buf[k] = byte(0)
		}
		size, err := conn.Read(buf)
		if err != nil {
			log.Errorf("%s cluster server连接发生错误: %v", conn.RemoteAddr().String(), err)
			server.onClose(cnode)
			conn.Close()
			return
		}
		log.Debugf("cluster server收到消息 %d 字节：%d %s", size, buf[:size], string(buf[:size]))
		server.onMessage(cnode, buf[:size])
	}
}

func (server *tcp_server) onClose(conn *tcp_client_node) {
	server.lock.Lock()
	for index, client := range server.clients {
		if client.conn == conn.conn {
			client.is_connected = false
			server.clients_count--
			log.Warnf("cluster server客户端掉线 %s", (*client.conn).RemoteAddr().String())
			server.clients = append(server.clients[:index], server.clients[index+1:]...)
			break
		}
	}
	server.lock.Unlock()
}

func (server *tcp_server) onMessage(conn *tcp_client_node, msg []byte) {
	conn.recv_buf.Write(msg)
	for {
		clen := conn.recv_buf.Size()
		if clen < 6 {
			return
		}
		cmd, _       := conn.recv_buf.ReadInt16() // 2字节 command
		client_id, _ := conn.recv_buf.Read(32)    // 32字节的client_id
		content      := make([]string, 1)
		log.Debugf("cluster server收到消息，cmd=%d %d %s", cmd, len(client_id), string(client_id))

		var index       = 0
		var content_len = 0
		var err error   = nil
		for {
			content_len, err = conn.recv_buf.ReadInt32()
			if err != nil {
				break
			}
			cb, err := conn.recv_buf.Read(content_len)
			if err != nil {
				break
			}
			content = append(content[:index], string(cb))
			log.Debugf("cluster server收到消息content=%d %d %s %v", content_len, index, content[index], cb)
			index++
		}

		switch cmd {
		case CMD_APPEND_NET:
			log.Infof("cluster server收到追加网络节点消息")

			//断开当前的c端连接
			//current_node.client.close()
			////c端连接的追加节点的s端
			//current_node.client.reset("ip", /*"port"*/0)
			//current_node.client.connect()
			////给连接到的s端发送一个指令，让s端的c端连接的第一个节点的s端
			//current_node.client.send(0, "请链接到第一个节点的s端")

			//4字节 weight
			//weight := int(conn.recv_buf[6]) +
			//    int(conn.recv_buf[7] << 8) +
			//    int(conn.recv_buf[8] << 16) +
			//    int(conn.recv_buf[9] << 32)

			//log.Println("weight：", weight)
			//心跳包
		case CMD_APPEND_NODE:
			log.Infof("cluster server收到追加链表节点消息")
			//转发给自己的client端
			//(*conn.conn).Write(server.pack(CMD_APPEND_NODE, ""))

			ip     := content[0]
			port, _:= strconv.Atoi(content[1])
			server.client.reset(ip, port)
			if server.client.connect() {
				server.client.send(CMD_APPEND_NODE_SURE, []string{
					server.service_ip,
					fmt.Sprintf("%d", server.port),
				})
			}


		case CMD_CONNECT_FIRST:
			//log.Println("cluster server收到追加闭环消息：", content[0] + ":" + content[1])
			//server.client.close()
			//port, _:= strconv.Atoi(content[1])
			//server.client.reset(content[0], port)
			//server.client.connect()
		case CMD_APPEND_NODE_SURE:
			ip     := content[0]
			port, _:= strconv.Atoi(content[1])
			server.cluster.enableNode(ip, port)
		default:
			//server.send(cmd, client_id, content)
		}
		conn.recv_buf.ResetPos()
	}
}
