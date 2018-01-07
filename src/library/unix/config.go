package unix

import (
	"context"
	"library/binlog"
	"net"
)

const (
	CMD_STOP         = 1
	CMD_RELOAD       = 2
	CMD_JOINTO       = 3
	CMD_SHOW_MEMBERS = 4
)

type UnixClient struct {
	addr string
	conn *net.Conn
}

type UnixServer struct {
	addr    string
	cancel  *context.CancelFunc
	binlog  *binlog.Binlog
	pidFile string
	cluster *binlog.TcpServer
}
