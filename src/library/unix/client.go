package unix

import (
	"github.com/juju/errors"
	log "github.com/jilieryuyi/logrus"
	"library/file"
	"net"
	"library/path"
)

func NewUnixClient() *UnixClient {
	addr := path.CurrentPath + "/wing-binlog-go.sock"
	client := &UnixClient{
		addr: addr,
	}
	client.Start()
	return client
}

func (client *UnixClient) Read() ([]byte, error) {
	if client.conn == nil {
		return nil, errors.New("not connect")
	}
	buf := make([]byte, 10240)

	n, err := (*client.conn).Read(buf[:])
	if err != nil {
		return nil, err
	}
	//log.Debugf("unix服务客户端收到消息：%s", string(buf[0:n]))
	return buf[:n], nil
}

func (client *UnixClient) Start() {
	client.conn = nil
	socketFile := &file.WFile{client.addr}
	if !socketFile.Exists() {
		return
	}
	c, err := net.Dial("unix", client.addr)
	if err != nil {
		log.Panicf("unix服务客户端异常：%+v", err)
	}
	client.conn = &c
	//go client.onMessage()
}

func (client *UnixClient) Pack(cmd int, msg string) []byte {
	m := []byte(msg)
	l := len(m)
	r := make([]byte, l+6)
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
	if client.conn == nil {
		return 0
	}
	n, err := (*client.conn).Write(msg)
	if err != nil {
		log.Error("unix服务客户端Write异常：%+v", err)
	}
	return n
}

func (client *UnixClient) Close() {
	if client.conn == nil {
		return
	}
	(*client.conn).Close()
}
