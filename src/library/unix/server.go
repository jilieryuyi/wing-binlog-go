package unix

import (
	"net"
	log "github.com/sirupsen/logrus"
	"library/file"
)

func NewUnixServer() *UnixServer {
	addr := file.GetCurrentPath() + "/wing-binlog-go.sock"
	server := &UnixServer{
		addr : addr,
	}
	return server
}

//func init() {
//	server := NewUnixServer()
//	server.Start()
//}

func (server *UnixServer) onConnect(c net.Conn) {
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}
		data := buf[0:nr]
		log.Debugf("unix服务收到消息：%s", string(data))
		_, err = c.Write(data)
		if err != nil {
			log.Errorf("unix服务Write异常：%+v ", err)
		}
	}
}

func (server *UnixServer) Start() {
	go func() {
		log.Debug("unix服务启动，等待新的连接...")
		listen, err := net.Listen("unix", server.addr)
		if err != nil {
			log.Panicf("unix服务异常：%+v", err)
		}
		for {
			fd, err := listen.Accept()
			if err != nil {
				log.Panicf("unix服务异常：%+v", err)
			}
			go server.onConnect(fd)
		}
	}()
}
