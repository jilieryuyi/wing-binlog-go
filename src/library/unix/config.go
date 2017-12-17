package unix

import (
	"net"
	"context"
	"library/binlog"
	"library/cluster"
)
const (
	CMD_STOP = 1
	CMD_RELOAD = 2
	CMD_JOINTO = 3
)
type UnixClient struct {
	addr string
	conn *net.Conn
}

type UnixServer struct {
	addr string
	cancel *context.CancelFunc
	binlog *binlog.Binlog
	pidFile string
	cluster *cluster.TcpServer
}
