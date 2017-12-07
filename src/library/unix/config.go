package unix

import "net"

type UnixClient struct {
	addr string
	conn *net.Conn
}

type UnixServer struct {
	addr string
}
