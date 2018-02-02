package unix

import (
	"bytes"
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"library/binlog"
	"library/file"
	"net"
	"os"
	"library/path"
)

func NewUnixServer() *UnixServer {
	addr := path.CurrentPath + "/wing-binlog-go.sock"
	server := &UnixServer{
		addr: addr,
	}
	return server
}

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
			int(data[3] << 24)
		cmd := 	int(data[4]) +
			int(data[5] << 8)

		content := bytes.ToLower(data[6:])

		log.Debugf("unix服务收到消息：%s, %+v, %d", string(data), data, length)

		switch cmd {
		case CMD_STOP:
			log.Debug("收到退出指令，程序即将退出")
			server.clear()
			(*server.cancel)()
			server.binlog.Close()
			fmt.Println("服务退出...")
			os.Exit(0)
		case CMD_RELOAD:
			{
				log.Debugf("收到重新加载指令：%s", string(content))
				server.binlog.Reload(string(content))
			}
		case CMD_JOINTO:
			log.Debugf("收到加入群集指令：%s", string(content))
		case CMD_SHOW_MEMBERS:
			members := server.binlog.Drive.GetMembers()
			if members != nil {
				hostname, err := os.Hostname()
				if err != nil {
					hostname = ""
				}
				l := len(members)
				res := fmt.Sprintf("current node: %s\r\n", hostname)
				res += fmt.Sprintf("cluster size: %d node(s)\r\n", l)
				res += fmt.Sprintf("======+=========================+================================+===============\r\n")
				res += fmt.Sprintf("%-6s| %-23s | %-30s | %-8s | %s\r\n", "index", "node", "session", "role", "status")
				res += fmt.Sprintf("------+-------------------------+----------+---------------\r\n")
				for i, member := range members {
					role := "follower"
					if member.IsLeader {
						role = "leader"
					}
					res += fmt.Sprintf("%-6d| %-23s | %-30s | %-8s | %s\r\n", i, member.Hostname, member.Session, role, member.Status)
				}
				res += fmt.Sprintf("------+-------------------------+----------+---------------\r\n")
				c.Write([]byte(res))
			} else {
				c.Write([]byte("no members found"))
			}
		case CMD_CLEAR:
			//server.binlog.Drive.ClearOfflineMembers()
			//c.Write([]byte("ok"))
		default:
			log.Error("不支持的指令：%d：%s", cmd, string(content))
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
	f = file.WFile{server.pidFile}
	if f.Exists() {
		f.Delete()
	}
}

func (server *UnixServer) Close() {
	server.clear()
}

func (server *UnixServer) Start(binlog *binlog.Binlog, cancel *context.CancelFunc, pid string) {
	server.cancel = cancel
	server.binlog = binlog
	server.pidFile = pid
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
