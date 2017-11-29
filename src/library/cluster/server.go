package cluster

import (
	"fmt"
	"net"
	"time"
	log "github.com/sirupsen/logrus"
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
func (server *tcp_server) send(cmd int, msg []string){
	// todo 这里需要加入channel和select，超时机制
	server.lock.Lock()
	log.Println("---clients", server.clients)
	for _, client := range server.clients {
		(*client.conn).Write(server.pack(cmd, msg))
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
		recv_buf           : make([]byte, TCP_RECV_DEFAULT_SIZE),
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

func (tcp *tcp_server) pack(cmd int, msgs []string) []byte {
	l := 0
	for _,msg := range msgs {
		l += len([]byte(msg)) + 4
	}

	// l+2 为实际的包内容长度，前缀4字节存放包长度
	r := make([]byte, l + 6)

	// l+2 为实际的包内容长度
	cl := l + 2

	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 32)

	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)

	base_start := 6
	for _, msg := range msgs {
		m  := []byte(msg)
		ml := len(m)
		// 前4字节存放长度
		r[base_start + 0] = byte(ml)
		r[base_start + 1] = byte(ml >> 8)
		r[base_start + 2] = byte(ml >> 16)
		r[base_start + 3] = byte(ml >> 32)
		base_start += 4
		// 实际的内容
		copy(r[base_start:], m)
		base_start += len(m)
	}

	return r
}



func (server *tcp_server) onMessage(conn *tcp_client_node, msg []byte, size int) {
	conn.recv_buf = append(conn.recv_buf[:conn.recv_bytes - size], msg[0:size]...)

	for {
		clen := len(conn.recv_buf)
		if clen < 6 {
			return
		} else if clen > TCP_RECV_DEFAULT_SIZE {
			// 清除所有的读缓存，防止发送的脏数据不断的累计
			conn.recv_buf = make([]byte, TCP_RECV_DEFAULT_SIZE)
			log.Println("cluster server新建缓冲区")
			return
		}

		//4字节长度
		content_len := int(conn.recv_buf[0]) +
			int(conn.recv_buf[1] << 8) +
			int(conn.recv_buf[2] << 16) +
			int(conn.recv_buf[3] << 32)

		//2字节 command
		cmd := int(conn.recv_buf[4]) + int(conn.recv_buf[5] << 8)
		content := conn.recv_buf[6: content_len + 4]

		log.Println("cluster server收到消息，cmd=", cmd, "content=", string(content))

		//log.Println("content：", conn.recv_buf)
		//log.Println("content_len：", content_len)
		//log.Println("cmd：", cmd)
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
			server.send(cmd, []string{string(content)})
			//conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
		}

		//数据移动
		//log.Println(content_len + 4, conn.recv_bytes)
		conn.recv_buf = append(conn.recv_buf[:0], conn.recv_buf[content_len + 4:conn.recv_bytes]...)
		conn.recv_bytes = conn.recv_bytes - content_len - 4
		//log.Println("移动后的数据：", conn.recv_bytes, len(conn.recv_buf), string(conn.recv_buf))
	}
}
