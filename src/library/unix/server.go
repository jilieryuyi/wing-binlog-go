package unix

import (
	"bytes"
	"fmt"
	log "github.com/jilieryuyi/logrus"
	"library/file"
	"net"
	"os"
	"library/path"
	"library/app"
	"library/binlog"
)

func NewUnixServer(ctx *app.Context, binlog *binlog.Binlog) *UnixServer {
	addr := path.CurrentPath + "/wing-binlog-go.sock"
	server := &UnixServer{
		addr: addr,
		ctx : ctx,
		binlog:binlog,
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
		log.Debugf("unix receive：%s, %+v, %d", string(data), data, length)
		switch cmd {
		case CMD_STOP:
			log.Debug("get stop cmd, app will stop later")
			server.clear()
			server.ctx.Cancel()
			server.binlog.Close()
			fmt.Println("service exit...")
			os.Exit(0)
		case CMD_RELOAD:
			log.Debugf("收到重新加载指令：%s", string(content))
			server.binlog.Reload(string(content))
		case CMD_JOINTO:
			log.Debugf("收到加入群集指令：%s", string(content))
		case CMD_SHOW_MEMBERS:
			members := server.binlog.GetMembers()
			if members != nil {
				hostname, err := os.Hostname()
				if err != nil {
					hostname = ""
				}
				l := len(members)
				res := fmt.Sprintf("current node: %s\r\n", hostname)
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
	f = file.WFile{server.ctx.PidFile}
	if f.Exists() {
		f.Delete()
	}
}

func (server *UnixServer) Close() {
	server.clear()
}

func (server *UnixServer) Start() {
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
