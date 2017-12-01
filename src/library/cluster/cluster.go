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
        Listen      : config.Cluster.Listen,
        Port        : config.Cluster.Port,
        ServiceIp   : config.Cluster.ServiceIp,
        is_down     : false,
        nodes       : make([]*cluster_node, CLUSTER_NODE_DEFAULT_SIZE),
        nodes_count : 0,
        lock        : new(sync.Mutex),
    }
    // 将自身设置为第一个节点
    node.nodes[0] = &cluster_node{
        service_ip:config.Cluster.ServiceIp,
        port:config.Cluster.Port,
        is_enable:true,
    }
    node.nodes_count++
    node.client   = &tcp_client {
        ip         : config.Cluster.ServiceIp,
        port       : config.Cluster.Port,
        is_closed  : true,
        recv_times : 0,
        recv_buf   : buffer.NewBuffer(TCP_RECV_DEFAULT_SIZE),
        client_id  : util.RandString(),
        lock       : new(sync.Mutex),
    }
    // 分布式节点的s端
    node.server = &tcp_server{
        listen        : config.Cluster.Listen,
        service_ip    : config.Cluster.ServiceIp,
        port          : config.Cluster.Port,
        cluster       : node,
        client        : node.client,
        clients       : make([]*tcp_client_node, 4),
        lock          : new(sync.Mutex),
        clients_count : 0,
    }
    node.server.start()
}

// 从当前节点发出添加节点邀请，把目标节点添加为分布式节点
func (cluster *Cluster) AddNode(ip string, port int) {
    cluster.lock.Lock()
    cluster.client.close()
    cluster.client.reset(ip, port)
    if cluster.client.connect() {
        cluster.client.send(CMD_APPEND_NODE, []string{
            cluster.ServiceIp,                            //当前节点服务ip
            fmt.Sprintf("%d", cluster.Port),        //当前节点的服务端口
            fmt.Sprintf("%d", cluster.nodes_count), //分配给目标节点的索引
        })
    }
    cluster.appendNode(ip, port)
    cluster.lock.Unlock()
}

// 收到AddNode的成功握手消息之后，执行此api，将节点添加到当前节点列表
func (cluster *Cluster) appendNode(ip string, port int){
    cluster.nodes[cluster.nodes_count] = &cluster_node{
        service_ip:ip,
        port:port,
        is_enable:false,
    }
    cluster.nodes_count++
}

func (cluster *Cluster) enableNode(ip string, port int) {
    cluster.lock.Lock()
    for _, node := range cluster.nodes {
        if node.service_ip == ip && node.port == port {
            node.is_enable = true
            break
        }
    }
    cluster.lock.Unlock()
}