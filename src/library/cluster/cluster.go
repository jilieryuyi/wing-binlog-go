package cluster

import (
    "fmt"
    "library/util"
   // "time"
    "log"
    "sync"
)

//全局-第一个节点和最后一个节点
var first_node   *Cluster
var last_node    *Cluster
var current_node *Cluster

// 初始化当前节点，
func init() {
    log.Println("cluster server初始化 ...............")
    log.Println("listen: 0.0.0.0:9990")
    //这里的参数应该是读取配置文件来的
    ip   := "0.0.0.0";
    port := 9990

    // 新建一个分布式节点
    first_node = NewCluster(ip, port)
    first_node.is_first = true
    first_node.is_last  = true

    // 分布式节点的c端，c端默认连接到下一个节点的s端
    // 也就是说s端默认连接的是上一个节点的c端
    // 所以，环形网络链表的第一个节点的s端连接着最后一个节点的c端
    first_node.client   = &tcp_client {
        ip         : "127.0.0.1",
        port       : 9990,
        is_closed  : false,
        recv_times : 0,
        recv_bytes : 0,
        recv_buf   : make([]byte, TCP_RECV_DEFAULT_SIZE),
        client_id  : util.RandString(),
    }

    // 分布式节点的s端
    first_node.server = &tcp_server{
        listen  : ip,
        port    : port,
        client  : first_node.client,
        clients : make([]*tcp_client_node, 4),
        lock    : new(sync.Mutex),
    }

    first_node.server.start(first_node.client)
    // c端连接
    //time.Sleep(time.Second)
    //first_node.client.connect()

    //debug
    first_node.client.send(99, []string{"hello word"})
    // s端服务开启
    current_node = first_node
    last_node    = first_node
}

// 新建一个群集节点
func NewCluster(ip string, port int) *Cluster {
    c := &Cluster {
        Ip : ip,
        Port:port,
    }
    c.next     = nil
    c.prev     = nil
    c.index    = 0
    c.is_first = false
    c.is_last  = false
    c.is_down  = false
    return c
}


func Append(c *Cluster) {

    //todo 追加节点还涉及到一个环形链表同步的问题
    //这里的client连接的就是最后一个节点的s端
    first_node.server.send(
        CMD_APPEND_NODE,
        []string{
            first_node.client.client_id,
            c.Ip,
            fmt.Sprintf("%s", c.Port),
        })

    // 以下为标准的链表操作，追加链表，首尾相连
    last := last_node
    last.next    = c
    last.is_last = false

    c.is_last = true
    c.prev    = last
    c.index   = last.index+1
    c.next    = first_node  //最后一个节点的下一个肯定是第一个节点
    first_node.prev = c

    // last_node 连接 c
    // 把之前的节点同步到最后的节点上
    last_node = c

    // last_node连接first_node
    // 然后整体就形成了一个tcp连接环
}

// 打印环形链表 -- 测试
func Print() {
    c1 := NewCluster("0.0.0.0", 9989);
    Append(c1);
    c2 := NewCluster("0.0.0.0", 9988);
    Append(c2);
    c3 := NewCluster("0.0.0.0", 9987);
    Append(c3);

    current := first_node

    for {
        fmt.Println(current.index, "=>", current.Ip, current.Port)
        current = current.next

        if current == first_node {
            fmt.Println(current.index, "=>", current.Ip, current.Port)
            break
        }
    }
}
