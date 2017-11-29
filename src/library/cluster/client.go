package cluster

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	log "github.com/sirupsen/logrus"
)


//type Msg struct {
//	Data string `json:"data"`
//	Type int    `json:"type"`
//}
//
//type Resp struct {
//	Data string `json:"data"`
//	Status int  `json:"status"`
//}



func (client *tcp_client) connect() {

	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", client.ip, client.port))
	if err != nil {
		fmt.Println("Error connecting:", err)
		os.Exit(1)
	}
	client.conn = &conn
	client.recv_bytes = 0;
	//defer conn.Close()
	//fmt.Println("Connecting to " + *host + ":" + *port)
	// 下面进行读写
	//var wg sync.WaitGroup
	//wg.Add(2)
	//go handleWrite(conn)

	go func(){
		var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
		for {
			buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
			//清空旧数据 memset
			for k,_:= range buf {
				buf[k] = byte(0)
			}
			size, err := conn.Read(buf)
			if err != nil {
				log.Println(conn.RemoteAddr().String(), "连接发生错误: ", err)
				client.onClose(&conn);
				client.close();
				return
			}
			log.Println("收到消息",size,"字节：", buf[:size], string(buf))
			atomic.AddInt64(&client.recv_times, int64(1))
			client.recv_bytes += int64(size)
			client.onMessage(buf, size)
		}
	}()
	//wg.Wait()
}

func (client *tcp_client) close() {
	if client.is_closed {
		return
	}
	client.is_closed = true
	(*client.conn).Close()
}

func (client *tcp_client) onClose(conn *net.Conn)  {

}

func (tcp *tcp_client) pack(cmd int, msgs []string) []byte {

	client_id_len := len(tcp.client_id)

	// 获取实际包长度
	l := 0
	for _,msg := range msgs {
		l += len([]byte(msg)) + 4
	}

	// l+2 为实际的包内容长度，前缀4字节存放包长度
	r := make([]byte, l + 6 + client_id_len)

	// l+2 为实际的包内容长度，2字节cmd
	cl := l + 2 + client_id_len

	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 32)

	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)

	copy(r[6:], []byte(tcp.client_id))

	base_start := 6 + client_id_len
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

func (client *tcp_client) reset(ip string, port int) {
	client.ip = ip
	client.port = port
}

func (client *tcp_client) send(cmd int, msgs []string) {
	(*client.conn).Write(client.pack(cmd, msgs))
	//defer wg.Done()
	// write 10 条数据
	//for i := 10; i > 0; i-- {
	//	d := "hello " + strconv.Itoa(i)
	//	msg := Msg{
	//		Data: d,
	//		Type: 1,
	//	}
	//	// 序列化数据
	//	b, _ := json.Marshal(msg)
	//	writer := bufio.NewWriter(conn)
	//	_, e := writer.Write(b)
	//	//_, e := conn.Write(b)
	//	if e != nil {
	//		fmt.Println("Error to send message because of ", e.Error())
	//		break
	//	}
	//	// 增加换行符导致server端可以readline
	//	//conn.Write([]byte("\n"))
	//	writer.Write([]byte("\n"))
	//	writer.Flush()
	//}
	//fmt.Println("Write Done!")
}

//func (client *tcp_client) onMessage(conn net.Conn) {
//	//defer wg.Done()
//	reader := bufio.NewReader(conn)
//	// 读取数据
//	for i := 1; i <= 10; i++ {
//		//line, err := reader.ReadString(byte('\n'))
//		line, _, err := reader.ReadLine()
//		if err != nil {
//			fmt.Print("Error to read message because of ", err)
//			return
//		}
//		// 反序列化数据
//		var resp Resp
//		json.Unmarshal(line, &resp)
//		fmt.Println("Status: ", resp.Status, " Content: ", resp.Data)
//	}
//	fmt.Println("Read Done!")
//}

func (client *tcp_client) onMessage(msg []byte, size int) {
	//如果收到自己发出去的消息，则终止转发

	client.recv_buf = append(client.recv_buf[:client.recv_bytes - int64(size)], msg[0:size]...)

	for {
		clen := len(client.recv_buf)
		if clen < 6 {
			return
		} else if clen > TCP_RECV_DEFAULT_SIZE {
			// 清除所有的读缓存，防止发送的脏数据不断的累计
			client.recv_buf = make([]byte, TCP_RECV_DEFAULT_SIZE)
			log.Println("新建缓冲区")
			return
		}

		//4字节长度
		content_len := int(client.recv_buf[0]) +
			int(client.recv_buf[1] << 8) +
			int(client.recv_buf[2] << 16) +
			int(client.recv_buf[3] << 32)

		//2字节 command
		cmd       := int(client.recv_buf[4]) + int(client.recv_buf[5] << 8)
		client_id := client.recv_buf[6: 38]
		content   := client.recv_buf[38: content_len + 4]

		log.Println("cluster client收到消息，cmd=", cmd, "content=", string(content), len(client_id), string(client_id))

		if string(client_id) == client.client_id {
			log.Println("cluster client收到消息闭环", string(client_id))
		}


		//log.Println("content：", conn.recv_buf)
		//log.Println("content_len：", content_len)
		//log.Println("cmd：", cmd)
		switch cmd {
		case CMD_APPEND_NODE:
			client.send(CMD_APPEND_NODE, []string{""})
		//case CMD_TICK:
			//log.Println("收到心跳消息")
			//conn.send_queue <- tcp.pack(CMD_TICK, "ok")
			//心跳包
		default:
			//conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
		}

		//数据移动
		//log.Println(content_len + 4, conn.recv_bytes)
		client.recv_buf = append(client.recv_buf[:0], client.recv_buf[content_len + 4:client.recv_bytes]...)
		client.recv_bytes = client.recv_bytes - int64(content_len) - 4
		//log.Println("移动后的数据：", conn.recv_bytes, len(conn.recv_buf), string(conn.recv_buf))
	}
}
