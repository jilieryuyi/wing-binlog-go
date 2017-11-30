package cluster

import (
	"fmt"
	"net"
	"time"
	log "github.com/sirupsen/logrus"
	"library/buffer"
)

func (server *tcp_server) start(client *tcp_client) {

	//建立socket，监听端口
	dns := fmt.Sprintf("%s:%d", server.listen, server.port)
	listen, err := net.Listen("tcp", dns)

	if err != nil {
		log.Println(err)
		return
	}
	go func() {
		defer listen.Close()
		log.Println("cluster server等待新的连接...")

		for {
			conn, err := listen.Accept()
			if err != nil {
				log.Println(err)
				continue
			}
			go server.onConnect(conn)
		}
	} ()
	client.connect()
}

// 广播
func (server *tcp_server) send(cmd int, client_id []byte, msg []string){
	// todo 这里需要加入channel和select，超时机制
	server.lock.Lock()
	log.Println("---clients", server.clients)
	for _, client := range server.clients {
		send_msg := pack(cmd, string(client_id), msg)
		log.Println("cluster server发送消息：", len(send_msg), send_msg, string(send_msg))
		(*client.conn).Write(send_msg)
	}
	server.lock.Unlock()
}

func (server *tcp_server) onConnect(conn net.Conn) {
	log.Println("cluster server新的连接：",conn.RemoteAddr().String())

	cnode := &tcp_client_node {
		conn               : &conn,
		is_connected       : true,
		send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
		send_failure_times : 0,
		weight             : 0,
		connect_time       : time.Now().Unix(),
		send_times         : int64(0),
		recv_buf           : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),//make([]byte, TCP_RECV_DEFAULT_SIZE),
		recv_bytes         : 0,
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
			log.Println(conn.RemoteAddr().String(), "cluster server连接发生错误: ", err)
			server.onClose(cnode)
			conn.Close()
			return
		}
		log.Println("cluster server收到消息",size,"字节：", buf[:size], string(buf))
		cnode.recv_bytes += size
		server.onMessage(cnode, buf, size)
	}
}

func (server *tcp_server) onClose(conn *tcp_client_node) {
	//当前节点的 prev 挂了（conn为当前节点的prev）
	//conn的prev 跳过断开的节点 直接连接到当前节点
	//给 （conn的prev）发消息触发连接
	//把当前节点的prev节点标志位已下线
	server.lock.Lock()
	for index, client := range server.clients {
		if client.conn == conn.conn {
			client.is_connected = false
			server.clients_count--
			log.Println("cluster server客户端掉线", (*client.conn).RemoteAddr().String())
			server.clients = append(server.clients[:index], server.clients[index+1:]...)
			break
		}
	}
	server.lock.Unlock()
}

func (server *tcp_server) onMessage(conn *tcp_client_node, msg []byte, size int) {
	conn.recv_buf.Write(msg[0:size])
	for {
		clen := conn.recv_buf.Size()
		if clen < 6 {
			return
		}
		// 2字节 command
		cmd, _ := conn.recv_buf.ReadInt16()
		// 32字节的client_id
		client_id, _ := conn.recv_buf.Read(32)
		content := make([]string, 1)
		log.Println("cluster server收到消息，cmd=", cmd, len(client_id), string(client_id))
		index := 0
		for {
			l, err := conn.recv_buf.ReadInt32()
			if err != nil {
				break
			}
			cb, err := conn.recv_buf.Read(l)
			if err != nil {
				break
			}
			content = append(content[:index], string(cb))
			log.Println("cluster server收到消息content=", l, index, content[index], cb)
			index++
		}

		switch cmd {
		case CMD_APPEND_NET:
			log.Println("cluster server收到追加网络节点消息")

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
			log.Println("cluster server收到追加链表节点消息")
			//转发给自己的client端
			//(*conn.conn).Write(server.pack(CMD_APPEND_NODE, ""))
		default:
			server.send(cmd, client_id, content)
		}
		conn.recv_buf.ResetPos()
	}
}
