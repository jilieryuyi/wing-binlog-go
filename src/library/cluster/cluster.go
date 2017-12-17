package cluster

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"library/buffer"
)

func NewCluster() *TcpServer {
	config, _:= getServiceConfig()
	log.Infof("cluster server初始化 ...............")
	log.Infof("listen: %s:%d", config.Cluster.Listen, config.Cluster.Port)

	server := &TcpServer{
		listen : config.Cluster.Listen,
		port : config.Cluster.Port,
		lock : new(sync.Mutex),
		clients_count : 0,
		listener : nil,
		clients:make([]*tcpClientNode, 4),
	}

	server.Client = &tcpClient{
		isClosed : true,
		recvTimes : int64(0),
		recvBuf : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),
		lock : new(sync.Mutex),
	}

	return server
}