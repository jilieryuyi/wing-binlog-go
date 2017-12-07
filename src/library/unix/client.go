package unix

import (
	"net"
	"library/file"
	log "github.com/sirupsen/logrus"
)



func NewUnixClient() *UnixClient {
	addr := file.GetCurrentPath() + "/wing-binlog-go.sock"
	client := &UnixClient{
		addr : addr,
	}
	return client
}

func (client *UnixClient) onMessage() {
	buf := make([]byte, 1024)
	for {
		n, err := (*client.conn).Read(buf[:])
		if err != nil {
			return
		}
		log.Debugf("unix服务客户端收到消息：%s", string(buf[0:n]))
	}
}

func (client *UnixClient) Start() {
	c, err := net.Dial("unix", client.addr)
	if err != nil {
		log.Panicf("unix服务客户端异常：%+v", err)
	}
	client.conn = &c
	go client.onMessage()
}

func (client *UnixClient) Send(msg []byte) int {
	n, err := (*client.conn).Write(msg)
	if err != nil {
		log.Error("unix服务客户端Write异常：%+v", err)
	}
	return n
}

func (client *UnixClient) Close() {
	(*client.conn).Close()
}

