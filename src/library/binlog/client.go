package binlog

import (
	"net"
	"sync/atomic"
	log "github.com/sirupsen/logrus"
	"time"
	"fmt"
	"library/buffer"
)

func (client *tcpClient) ConnectTo(dns string) bool {
	client.lock.Lock()
	defer client.lock.Unlock()

	log.Debugf("connect to: %s", dns)
	var index int
	dns, index = client.getLeaderDns(dns)
	if dns == "" {
		return false
	}
	client.binlog.setMember(dns, true, index)
	client.dns = dns
	if client.connect() != nil {
		return false
	}

	go func(){
		var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte
		for {
			buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
			//清空旧数据 memset
			for i := range buf {
				buf[i] = byte(0)
			}
			size, err := (*client.conn).Read(buf)
			if err != nil {
				log.Errorf("%s 连接发生错误: %s", (*client.conn).RemoteAddr().String(), err)
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

func (client *tcpClient) connect() error {
	// 连接到leader
	conn, err := net.DialTimeout("tcp", client.dns, time.Second*3)
	if err != nil {
		log.Errorf("cluster服务client连接错误: %s", err)
		return err
	}
	if !client.isClosed {
		client.onClose()
	}
	client.conn = &conn
	client.isClosed = false

	//发送一个握手消息，用来确认加入集群
	dns := fmt.Sprintf("%s:%d", client.ServiceIp, client.ServicePort)
	log.Debugf("send join: %s", dns)
	client.Send(CMD_JOIN, dns)
	return nil
}

func (client *tcpClient) getLeaderDns(dns string) (string, int) {
	// todo 查询当前leader serviceIp
	conn, err := net.DialTimeout("tcp", dns, time.Second*3)
	if err != nil {
		log.Errorf("get leader dns error: %+v", err)
		return "", 0
	}
	buf := make([]byte, 256)
	sendMsg := client.pack(CMD_GET_LEADER, "")
	conn.Write(sendMsg)

	conn.SetReadDeadline(time.Now().Add(time.Second*3))
	size, err := conn.Read(buf)
	if err != nil || size <= 0 {
		log.Errorf("get leader service ip error: %+v", err)
		return "", 0
	}

	dataBuf := buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE)
	dataBuf.Write(buf[:size])
	clen, _ := dataBuf.ReadInt32()
	dataBuf.ReadInt16() // 2字节 command
	index, _ := dataBuf.ReadInt16()
	content, _ := dataBuf.Read(clen-4)
	log.Debugf("get leader is: %s--%d", string(content), index)
	conn.Close()

	return string(content), index
}

func (client *tcpClient) onClose()  {
	client.lock.Lock()
	defer client.lock.Unlock()

	if !client.isClosed {
		client.isClosed = true
		(*client.conn).Close()
	}

	log.Debug("cluster client close")
	//todo
	//如果当前节点数量只有两个
	//nodesCount := len(client.binlog.members)

	//todo 需要等待 ceil(n/2) 个节点确认（选举）
	//本身即是其中一个，所以等待int(n/2)个确认选举就可以了

	//1、得到next leader
	//2、如果当前节点不是next leader，则连接leader，确认选举
	//3、如果当前是next leader，如果数量达到int(n/2)则选举成功

	//集群leader当节点<=2的时候，有一个明显的缺点，就是如果两个节点的网络断开，
	//没办法进一步确认是否正的原leader节点已经下线，所以可能出现两个leader的情况
	//并非100%可靠

	if /*nodesCount <= 2 &&*/ client.binlog.isNextLeader() {
		log.Debug("current is next leader")
		//尝试重连三次
		errTimes := 0
		for i := 0; i < 3; i++ {
			err := client.connect()
			log.Debugf("try to reconnect %d times", (i+1))
			if err == nil {
				break
			} else {
				errTimes++
			}
		}
		if errTimes >= 3 {
			log.Debug("reconnect failure, set current node is leader")
			//如果都失败，则把当前节点设置为leader
			client.binlog.StartService()
			client.binlog.leader(true)
		}
	}

	//如果当前节点数量大于2
	//如果当前节点的索引为leader的索引的下一个
	//等待下一个节点确认leader断线--双确认
	//则将当前节点设置为leader
}

// todo 这里应该使用新的channel服务进行发送
func (client *tcpClient) Send(cmd int, msg string) {
	sendMsg := client.pack(cmd, msg)
	log.Debugf("cluster client发送消息, %d, %s", len(sendMsg), sendMsg)
	(*client.conn).Write(sendMsg)
}

func (client *tcpClient) pack(cmd int, msg string) []byte {
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
		size := client.recvBuf.Size()
		log.Debugf("cluster client buf size %s", size)
		if size < 6 {
			return
		}
		clen, _ := client.recvBuf.ReadInt32()
		// 2字节 command
		cmd, _     := client.recvBuf.ReadInt16()
		content, _ := client.recvBuf.Read(clen-2)
		log.Debugf("cluster服务client收到消息content=%d, %d, %s", cmd, clen, content)

		switch cmd {
			case CMD_POS:
				log.Debugf("cluster服务-client-binlog写入缓存：%s", string(content))
				client.binlog.BinlogHandler.SaveBinlogPostionCache(string(content))
				client.binlog.BinlogHandler.setCacheInfo(string(content))
			case CMD_JOIN:
				log.Debugf("cluster服务-client收到握手回复，加入群集成功")
				//这里是follower节点，所以后续要停止数据采集操作
				client.binlog.StopService(false)
				client.binlog.leader(false)
		    case CMD_NEW_NODE:
				//index := len(client.binlog.members) + 1
				// leader分配的索引
				index := int(content[0]) + int(content[1] << 8)
				dns := string(content[2:])
				client.binlog.setMember(dns, false, index)
			case CMD_KEEPALIVE:
				// keep alive
				log.Debugf("keep alive")
			default:
		}
		client.recvBuf.ResetPos()
	}
}

