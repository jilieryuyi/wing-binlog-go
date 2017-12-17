package binlog

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

	//发送一个握手消息，用来确认加入集群
	client.Send(CMD_JOIN, "")

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
func (client *tcpClient) Send(cmd int, message string) {
	sendMsg := client.pack(cmd, message)
	log.Debugf("cluster client发送消息, %d, %s", len(sendMsg), sendMsg)
	(*client.conn).Write(sendMsg)
}

func (tcp *tcpClient) pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m)
	r := make([]byte, l + 6)
	cl := l + 2
	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 32)
	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], m)
	return r
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
		log.Debugf("cluster服务client收到消息content=%d, %d, %s", cmd, contentLen, content)

		switch cmd {
			case CMD_POS:
				log.Debugf("cluster服务-client-binlog写入缓存：%s", string(content))
				client.binlog.BinlogHandler.SaveBinlogPostionCache(string(content))
			case CMD_JOIN:
				log.Debugf("cluster服务-client收到握手回复，加入群集成功")
				//这里是follower节点，所以后续要停止数据采集操作
				//global.GetBinlog().StopService()
			default:
		}
		client.recvBuf.ResetPos()
	}
}
