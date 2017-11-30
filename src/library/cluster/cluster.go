package cluster

import (
    "library/util"
    log "github.com/sirupsen/logrus"
    "sync"
    "library/buffer"
    "fmt"
)

func init() {
    config, _:= getServiceConfig()
    log.Println("cluster server初始化 ...............")
    log.Println("listen: ", fmt.Sprintf("%s:%d", config.Cluster.Listen, config.Cluster.Port))
    // 新建一个分布式节点
    node := &Cluster {
        Listen    : config.Cluster.Listen,
        Port      : config.Cluster.Port,
        ServiceIp : config.Cluster.ServiceIp,
        is_down   : false,
    }
    node.client   = &tcp_client {
        ip         : config.Cluster.ServiceIp,
        port       : config.Cluster.Port,
        is_closed  : false,
        recv_times : 0,
        recv_buf   : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),
        client_id  : util.RandString(),
        lock       : new(sync.Mutex),
    }
    // 分布式节点的s端
    node.server = &tcp_server{
        listen        : config.Cluster.Listen,
        port          : config.Cluster.Port,
        client        : node.client,
        clients       : make([]*tcp_client_node, 4),
        lock          : new(sync.Mutex),
        clients_count : 0,
    }
    node.server.start()
}