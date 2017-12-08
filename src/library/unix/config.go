package unix

import "net"
const (
	CMD_STOP = 1
)
type UnixClient struct {
	addr string
	conn *net.Conn
}

type UnixServer struct {
	addr string
}
