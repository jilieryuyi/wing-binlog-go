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

func (tcp *UnixClient) Pack(cmd int, msg string) []byte {
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

