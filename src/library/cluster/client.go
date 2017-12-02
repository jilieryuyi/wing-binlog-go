package cluster

import (
	"fmt"
	"net"
	"sync/atomic"
	log "github.com/sirupsen/logrus"
	//"strconv"
	"strconv"
)

func (client *tcp_client) connect() bool {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", client.ip, client.port))
	if err != nil {
		log.Errorf("Error connecting: %s", err)
		return false
	}
	client.lock.Lock()
	client.conn = &conn
	client.is_closed = false
	client.lock.Unlock()
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
				log.Errorf("%s 连接发生错误: %s", conn.RemoteAddr().String(), err)
				client.lock.Lock()
				client.onClose(&conn);
				if !client.is_closed {
					client.close();
				}
				client.lock.Unlock()
				return
			}
			log.Debugf("cluster client 收到消息 %d 字节：%d %s",  size, buf[:size],  string(buf[:size]))
			atomic.AddInt64(&client.recv_times, int64(1))
			client.onMessage(buf[:size])
		}
	}()
	return true
}

func (client *tcp_client) close() {
	client.lock.Lock()
	defer client.lock.Unlock()
	if client.is_closed {
		return
	}
	client.is_closed = true
	(*client.conn).Close()
}

func (client *tcp_client) onClose(conn *net.Conn)  {

}

func (client *tcp_client) reset(ip string, port int) {
	client.lock.Lock()
	defer client.lock.Unlock()
	client.ip   = ip
	client.port = port
}

// todo 这里应该使用新的channel服务进行发送
func (client *tcp_client) send(cmd int, msgs []string) {
	send_msg := pack(cmd, client.client_id, msgs)
	log.Println("cluster client发送消息", len(send_msg), string(send_msg), send_msg)
	(*client.conn).Write(send_msg)
}

// 收到消息回调函数
func (client *tcp_client) onMessage(msg []byte) {
	client.recv_buf.Write(msg)
	for {
		clen := client.recv_buf.Size()
		log.Debugf("cluster client buf size %s", clen)
		if clen < 6 {
			return
		}
		// 2字节 command
		cmd, _         := client.recv_buf.ReadInt16()
		client_id, _   := client.recv_buf.Read(32)
		content        := make([]string, 1)
		index          := 0
		log.Debugf("cluster client收到消息，cmd=%d %d %s", cmd, len(client_id), string(client_id))
		for {
			// 4字节长度，即int32
			l, err := client.recv_buf.ReadInt32()
			if err != nil {
				break
			}
			cb, err := client.recv_buf.Read(l)
			if err != nil {
				break
			}
			content = append(content[:index], string(cb))
			log.Debugf("cluster client收到消息content=%d %d %s %v", l, index, content[index], cb)
			index++
		}

		if string(client_id) == client.client_id {
			log.Debugf("cluster client收到消息闭环 %s", string(client_id))
			client.recv_buf.ResetPos()
			return
		}

		switch cmd {
			case CMD_APPEND_NODE:
				//client.send(CMD_APPEND_NODE, []string{""})
				log.Debugf("cluster client收到追加节点消息")
				log.Debugf("cluster client追加节点：%s", content[0] + ":" + content[1])

				// 关闭client连接
				client.close()
				// 将client连接到下一个节点
				port, _:= strconv.Atoi(content[1])
				client.reset(content[0], port)
				client.connect()
				//发送闭环指令
				//client.send(CMD_CONNECT_FIRST, []string{
				//	"10.0.33.75",
				//	fmt.Sprintf("%d", first_node.Port),
				//})

			default:
				//链路转发
				client.send(cmd, content)
		}
		client.recv_buf.ResetPos()
	}
}
