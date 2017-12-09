package unix

import (
	"net"
	log "github.com/sirupsen/logrus"
	"library/file"
	"os"
	"context"
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

		length := int(data[0]) +
			int(data[1] << 8) +
			int(data[2] << 16) +
			int(data[3] << 32)
		cmd := 	int(data[4]) +
			int(data[5] << 8)

		log.Debugf("unix服务收到消息：%s, %+v, %d", string(data), data, length)

		switch cmd {
		case CMD_STOP:
			log.Debug("收到退出指令，程序即将退出")
			server.clear()
			os.Exit(0)
		}

		//_, err = c.Write(data)
		//if err != nil {
		//	log.Errorf("unix服务Write异常：%+v ", err)
		//}
	}
}
func (server *UnixServer) clear() {
	f := file.WFile{server.addr}
	if f.Exists() {
		f.Delete()
	}
}

func (server *UnixServer) Start(ctx *context.Context) {
	server.ctx = ctx
	server.clear()
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
