package unix

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"library/file"
	"net"
	"os"
	"library/app"
	"library/binlog"
	"library/platform"
)

func NewUnixServer(ctx *app.Context, binlog *binlog.Binlog) *UnixServer {
	server := &UnixServer{
		addr: app.SockFile,
		ctx : ctx,
		binlog:binlog,
	}
	log.Debugf("unix server: %+v", *server)
	return server
}

func (server *UnixServer) onConnect(c net.Conn) {
	for {
		buf := make([]byte, 512)
		nr, err := c.Read(buf)
		if err != nil {
			return
		}
		data    := buf[0:nr]
		length  := int(data[0]) | int(data[1]) << 8 + int(data[2]) << 16 | int(data[3]) << 24
		cmd     := int(data[4]) | int(data[5]) << 8
		content := bytes.ToLower(data[6:])

		log.Debugf("unix receive：%s, %+v, %d", string(data), data, length)
		switch cmd {
		case CMD_STOP:
			log.Debug("get stop cmd, app will stop later")
			server.ctx.CancelChan <- struct{}{}
		case CMD_RELOAD:
			log.Debugf("receive reload cmd：%s", string(content))
			server.binlog.Reload(string(content))
		case CMD_SHOW_MEMBERS:
			members := server.binlog.GetMembers()
			currentIp, currentPort := server.binlog.GetCurrent()
			if members != nil {
				hostname, err := os.Hostname()
				if err != nil {
					hostname = ""
				}
				l := len(members)
				res := fmt.Sprintf("current node: %s(%s:%d)\r\n", hostname, currentIp, currentPort)
				res += fmt.Sprintf("cluster size: %d node(s)\r\n", l)
				res += fmt.Sprintf("======+==========================================+==========+===============\r\n")
				res += fmt.Sprintf("%-6s| %-40s | %-8s | %s\r\n", "index", "node", "role", "status")
				res += fmt.Sprintf("------+------------------------------------------+----------+---------------\r\n")
				for i, member := range members {
					role := "follower"
					if member.IsLeader {
						role = "leader"
					}
					res += fmt.Sprintf("%-6d| %-40s | %-8s | %s\r\n", i, fmt.Sprintf("%s(%s:%d)", member.Hostname, member.ServiceIp, member.Port), role, member.Status)
				}
				res += fmt.Sprintf("------+------------------------------------------+----------+---------------\r\n")
				c.Write([]byte(res))
			} else {
				c.Write([]byte("no members found"))
			}
		default:
			log.Errorf("does not support cmd：%d：%s", cmd, string(content))
		}
	}
}

func (server *UnixServer) clear() {
	if file.Exists(server.addr) {
		file.Delete(server.addr)
	}
}

func (server *UnixServer) Close() {
	server.clear()
}

func (server *UnixServer) Start() {
	server.clear()
	if !platform.System(platform.IS_WINDOWS) {
		go func() {
			log.Debug("unix service start")
			listen, err := net.Listen("unix", server.addr)
			if err != nil {
				log.Panicf("unix service error：%+v", err)
			}
			for {
				fd, err := listen.Accept()
				if err != nil {
					log.Panicf("unix service error：%+v", err)
				}
				go server.onConnect(fd)
			}
		}()
	}
}
