package cluster

import (
	"net"
	"sync/atomic"
	log "github.com/sirupsen/logrus"
)

func (client *tcpClient) ConnectTo(dns string) bool {
	conn, err := net.Dial("tcp", dns)
	if err != nil {
		log.Errorf("cluster服务client连接错误: %s", err)
		return false
	}
	if !client.isClosed {
		client.onClose()
	}
	client.lock.Lock()
	client.conn = &conn
	client.dns = dns
	client.isClosed = false
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
				client.onClose();
				return
			}
			log.Debugf("cluster client 收到消息 %d 字节：%d %s",  size, buf[:size],  string(buf[:size]))
			atomic.AddInt64(&client.recvTimes, int64(1))
			client.onMessage(buf[:size])
		}
	}()
	return true
}

func (client *tcpClient) onClose()  {
	client.lock.Lock()
	defer client.lock.Unlock()
	if client.isClosed {
		return
	}
	client.isClosed = true
	(*client.conn).Close()
}

// todo 这里应该使用新的channel服务进行发送
func (client *tcpClient) Send(cmd int, messages []string) {
	//send_msg := pack(cmd, client.client_id, msgs)
	//log.Println("cluster client发送消息", len(send_msg), string(send_msg), send_msg)
	//(*client.conn).Write(send_msg)
}

// 收到消息回调函数
func (client *tcpClient) onMessage(msg []byte) {
	client.recvBuf.Write(msg)
	for {
		clen := client.recvBuf.Size()
		log.Debugf("cluster client buf size %s", clen)
		if clen < 6 {
			return
		}
		// 2字节 command
		cmd, _         := client.recvBuf.ReadInt16()
		client_id, _   := client.recvBuf.Read(32)
		content        := make([]string, 1)
		index          := 0
		log.Debugf("cluster服务client收到消息，cmd=%d %d %s", cmd, len(client_id), string(client_id))
		for {
			// 4字节长度，即int32
			l, err := client.recvBuf.ReadInt32()
			if err != nil {
				break
			}
			cb, err := client.recvBuf.Read(l)
			if err != nil {
				break
			}
			content = append(content[:index], string(cb))
			log.Debugf("cluster client收到消息content=%d %d %s %v", l, index, content[index], cb)
			index++
		}

		//if string(client_id) == client.client_id {
		//	log.Debugf("cluster client收到消息闭环 %s", string(client_id))
		//	client.recvBuf.ResetPos()
		//	return
		//}

		switch cmd {
			case CMD_APPEND_NODE:
				//client.send(CMD_APPEND_NODE, []string{""})
				//log.Debugf("cluster client收到追加节点消息")
				//log.Debugf("cluster client追加节点：%s", content[0] + ":" + content[1])
				//
				//// 关闭client连接
				//client.close()
				//// 将client连接到下一个节点
				//port, _:= strconv.Atoi(content[1])
				//client.reset(content[0], port)
				//client.connect()
				//发送闭环指令
				//client.send(CMD_CONNECT_FIRST, []string{
				//	"10.0.33.75",
				//	fmt.Sprintf("%d", first_node.Port),
				//})

			default:
				//链路转发
				//client.send(cmd, content)
		}
		client.recvBuf.ResetPos()
	}
}
