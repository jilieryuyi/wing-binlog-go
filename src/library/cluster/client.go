package cluster

import (
	"net"
	"sync/atomic"
	log "github.com/sirupsen/logrus"
	"time"
)

func (client *tcpClient) ConnectTo(dns string) bool {
	client.lock.Lock()
	defer client.lock.Unlock()
	conn, err := net.DialTimeout("tcp", dns, time.Second*3)
	if err != nil {
		log.Errorf("cluster服务client连接错误: %s", err)
		return false
	}
	if !client.isClosed {
		client.onClose()
	}
	client.conn = &conn
	client.dns = dns
	client.isClosed = false
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
		contentLen, _  := client.recvBuf.ReadInt32()
		// 2字节 command
		cmd, _         := client.recvBuf.ReadInt16()
		content, _     := client.recvBuf.Read(contentLen-2)
		log.Debugf("cluster client收到消息content=%d, %d, %s", cmd, contentLen, content)

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
