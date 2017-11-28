package cluster

import (
    "fmt"
    "net"
    "log"
    "time"
)

//全局-第一个节点和最后一个节点
var first_node *Cluster
var last_node  *Cluster
var current_node *Cluster

// 初始化当前节点，
func init() {
    //这里的参数应该是读取配置文件来的
    ip   := "0.0.0.0";
    port := 9990
    first_node = NewCluster(ip, port)
    first_node.is_first = true
    first_node.is_last  = true
    first_node.client   = &tcp_client {
        ip         : "0.0.0.0",
        port       : 9990,
        is_closed  : false,
        recv_times : 0,
        recv_bytes : 0,
        recv_buf   : make([]byte, TCP_RECV_DEFAULT_SIZE),
    }
    first_node.client.connect()
    start(ip, port)
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

func sendAppend() {
    //第一个节点的c端发送一个append节点的指令到连接的s端
    first_node.client.send(CMD_APPEND_NODE, "")
    //todo 追加节点还涉及到一个环形链表同步的问题
}

func Append(c *Cluster) {

    sendAppend()

    if first_node != last_node {
        //断开last_node与first_node的连接
        //client to server
        //发送追加节点的操作到最后一个节点

    }

    last := last_node
    last.next = c
    last.is_last = false

    c.is_last = true
    c.prev    = last
    c.index   = last.index+1
    c.next    = first_node  //最后一个节点的下一个肯定是第一个节点
    first_node.prev = c

    //last_node 连接 c
    //把之前的节点同步到最后的节点上

    last_node = c

    //last_node连接first_node

    //然后整体就形成了一个tcp连接环
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

func onMessage(conn *tcp_client_node, msg []byte, size int) {
    //CMD_APPEND_NODE
    conn.recv_buf = append(conn.recv_buf[:conn.recv_bytes - size], msg[0:size]...)

    for {
        clen := len(conn.recv_buf)
        if clen < 6 {
            return
        } else if clen > TCP_RECV_DEFAULT_SIZE {
            // 清除所有的读缓存，防止发送的脏数据不断的累计
            conn.recv_buf = make([]byte, TCP_RECV_DEFAULT_SIZE)
            log.Println("新建缓冲区")
            return
        }

        //4字节长度
        content_len := int(conn.recv_buf[0]) +
            int(conn.recv_buf[1] << 8) +
            int(conn.recv_buf[2] << 16) +
            int(conn.recv_buf[3] << 32)

        //2字节 command
        cmd := int(conn.recv_buf[4]) + int(conn.recv_buf[5] << 8)

        //log.Println("content：", conn.recv_buf)
        //log.Println("content_len：", content_len)
        //log.Println("cmd：", cmd)
        switch cmd {
        case CMD_APPEND_NODE:
            log.Println("追加节点")

            //断开当前的c端连接
            current_node.client.close()
            //c端连接的追加节点的s端
            current_node.client.reset("ip", /*"port"*/0)
            current_node.client.connect()
            //给连接到的s端发送一个指令，让s端的c端连接的第一个节点的s端
            current_node.client.send(0, "请链接到第一个节点的s端")

            //4字节 weight
            //weight := int(conn.recv_buf[6]) +
            //    int(conn.recv_buf[7] << 8) +
            //    int(conn.recv_buf[8] << 16) +
            //    int(conn.recv_buf[9] << 32)

            //log.Println("weight：", weight)
            //心跳包
        default:
            //conn.send_queue <- tcp.pack(CMD_ERROR, fmt.Sprintf("不支持的指令：%d", cmd))
        }

        //数据移动
        //log.Println(content_len + 4, conn.recv_bytes)
        conn.recv_buf = append(conn.recv_buf[:0], conn.recv_buf[content_len + 4:conn.recv_bytes]...)
        conn.recv_bytes = conn.recv_bytes - content_len - 4
        //log.Println("移动后的数据：", conn.recv_bytes, len(conn.recv_buf), string(conn.recv_buf))
    }
}