package cluster

import (
    "fmt"
    "net"
    "log"
    "time"
)
const (
    TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
    TCP_DEFAULT_CLIENT_SIZE       = 64
    TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
    TCP_RECV_DEFAULT_SIZE         = 4096
    TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
)
type Cluster struct {
    Ip string     //节点ip
    Port int      //节点端口
    next *Cluster //下一个节点
    prev *Cluster //前一个节点
    is_first bool //是否为第一个
    is_last bool   //是否为最后一个
    is_down bool  //是否已下线
    index int
}

type tcp_client_node struct {
    conn *net.Conn           // 客户端连接进来的资源句柄
    is_connected bool        // 是否还连接着 true 表示正常 false表示已断开
    send_queue chan []byte   // 发送channel
    send_failure_times int64 // 发送失败次数
    weight int               // 权重 0 - 100
    recv_buf []byte          // 读缓冲区
    recv_bytes int           // 收到的待处理字节数量
    connect_time int64       // 连接成功的时间戳
    send_times int64         // 发送次数，用来计算负载均衡，如果 mode == 2
}

var __first_point *Cluster
var __last_point *Cluster

// 初始化当前节点，
func init() {
    //这里的参数应该是读取配置文件来的
    ip   := "127.0.0.1";
    port := 9990
    __first_point = NewCluster(ip, port)
    __first_point.is_first = true
    __first_point.is_last = true
    __last_point = __first_point
    start(ip, port)
}

func NewCluster(ip string, port int) *Cluster {
    c := &Cluster{
        Ip : ip,
        Port:port,
    }
    c.next = nil
    c.prev = nil
    c.index = 0
    c.is_first = false
    c.is_last = false
    c.is_down = false
    return c
}

func Append(c *Cluster) {

    if __first_point != __last_point {
        //断开__last_point与__first_point的连接
        //client to server
    }

    last := __last_point
    last.next = c
    last.is_last = false

    c.is_last = true
    c.prev    = last
    c.index   = last.index+1
    c.next    = __first_point  //最后一个节点的下一个肯定是第一个节点
    __first_point.prev = c

    //__last_point 连接 c
    //把之前的节点同步到最后的节点上

    __last_point = c

    //__last_point连接__first_point

    //然后整体就形成了一个tcp连接环
}

// 打印环形链表 -- 测试
func Print() {
    c1 := NewCluster("127.0.0.1", 9989);
    Append(c1);
    c2 := NewCluster("127.0.0.1", 9988);
    Append(c2);
    c3 := NewCluster("127.0.0.1", 9987);
    Append(c3);

    current := __first_point

    for {
        fmt.Println(current.index, "=>", current.Ip, current.Port)
        current = current.next

        if current == __first_point {
            fmt.Println(current.index, "=>", current.Ip, current.Port)
            break
        }
    }
}

func start(ip string, port int) {
    go func() {
        //建立socket，监听端口
        dns := fmt.Sprintf("%s:%d", ip, port)
        listen, err := net.Listen("tcp", dns)

        if err != nil {
            log.Println(err)
            return
        }

        defer func() {
            listen.Close();
        }()

        log.Println("cluster等待新的连接...")

        for {
            conn, err := listen.Accept()
            if err != nil {
                log.Println(err)
                continue
            }
            go onConnect(conn)
        }
    } ()
}

func onConnect(conn net.Conn) {
    log.Println("cluster新的连接：",conn.RemoteAddr().String())

    cnode := &tcp_client_node {
        conn               : &conn,
        is_connected       : true,
        send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
        send_failure_times : 0,
        weight             : 0,
        connect_time       : time.Now().Unix(),
        send_times         : int64(0),
        recv_buf           : make([]byte, TCP_RECV_DEFAULT_SIZE),
        recv_bytes         : 0,
    }

    var read_buffer [TCP_DEFAULT_READ_BUFFER_SIZE]byte

    // 设定3秒超时，如果添加到分组成功，超时限制将被清除
    conn.SetReadDeadline(time.Now().Add(time.Second*3))
    for {
        buf := read_buffer[:TCP_DEFAULT_READ_BUFFER_SIZE]
        //清空旧数据 memset
        for k,_:= range buf {
            buf[k] = byte(0)
        }
        size, err := conn.Read(buf)
        if err != nil {
            log.Println(conn.RemoteAddr().String(), "连接发生错误: ", err)
            onClose(cnode)
            conn.Close()
            return
        }
        log.Println("收到消息",size,"字节：", buf[:size], string(buf))
        cnode.recv_bytes += size
        onMessage(cnode, buf, size)
    }
}

func onClose(conn *tcp_client_node) {
    //当前节点的 prev 挂了（conn为当前节点的prev）

    //conn的prev 跳过断开的节点 直接连接到当前节点
    //给 （conn的prev）发消息触发连接

    //把当前节点的prev节点标志位已下线
}

func onMessage(node *tcp_client_node, buf []byte, size int) {

}