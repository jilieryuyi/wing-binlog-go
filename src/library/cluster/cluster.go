package cluster

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"library/buffer"
	"library/file"
	"context"
	"os"
)

func NewCluster(ctx *context.Context) *TcpServer {
	config, _:= getServiceConfig()
	log.Infof("cluster server初始化 ...............")
	log.Infof("listen: %s:%d", config.Cluster.Listen, config.Cluster.Port)

	server := &TcpServer{
		listen : config.Cluster.Listen,
		port : config.Cluster.Port,
		lock : new(sync.Mutex),
		clientsCount : 0,
		listener : nil,
		clients:make([]*tcpClientNode, 4),
		wg : new(sync.WaitGroup),
		sendFailureTimes : int64(0),
		ctx : ctx,
	}

	// 初始化缓存文件句柄
	mysql_binlog_position_cache := file.CurrentPath +"/cache/mysql_binlog_position.pos"
	dir := file.WPath{mysql_binlog_position_cache}
	dir = file.WPath{dir.GetParent()}
	dir.Mkdir()
	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC | os.O_TRUNC
	var err error
	server.cacheHandler, err = os.OpenFile(
		mysql_binlog_position_cache, flag , 0755)
	if err != nil {
		log.Panicf("cluster服务，打开缓存文件错误：%s, %+v", mysql_binlog_position_cache, err)
	}

	server.Client = &tcpClient{
		isClosed : true,
		recvTimes : int64(0),
		recvBuf : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),
		lock : new(sync.Mutex),
		cacheHandler:server.cacheHandler,
	}

	return server
}