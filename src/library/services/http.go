package services

import (
    "log"
    "errors"
    "time"
    "sync/atomic"
    "sync"
    "regexp"
    "library/http"
)

const HTTP_POST_TIMEOUT = 3 //3秒超时
const HTTP_CACHE_LEN = 10000
const HTTP_CACHE_BUFFER_SIZE = 4096

var ERR_STATUS error =  errors.New("错误的状态码")

type HttpService struct {
    send_queue chan []byte     // 发送channel
    groups [][]*httpNode       // 客户端分组
    groups_filter [][]string   // 分组过滤器
    lock *sync.Mutex           // 互斥锁，修改资源时锁定
    send_failure_times int64   // 发送失败次数
    enable bool
}

type httpNode struct {
    url string                  // url
    send_queue chan []byte      // 发送channel
    send_times int64            // 发送次数
    send_failure_times int64    // 发送失败次数
    is_down bool                // 是否因为故障下线的节点
    failure_times_flag int32    // 发送失败次数，用于配合last_error_time检测故障，故障定义为：连续三次发生错误和返回错误
    lock *sync.Mutex            // 互斥锁，修改资源时锁定

    cache [][]byte
    cache_index int
    cache_is_init bool
    cache_full bool
}

func NewHttpService(config *HttpConfig) *HttpService {
    if !config.Enable {
        return &HttpService{enable:config.Enable}
    }
    glen := len(config.Groups)
    client := &HttpService {
        send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
        lock               : new(sync.Mutex),
        groups             : make([][]*httpNode, glen),
        groups_filter      : make([][]string, glen),
        send_failure_times : int64(0),
        enable             : config.Enable,
    }
    index := 0
    for _, v := range config.Groups {
        l := len(v.Nodes)
        client.groups[index]      = make([]*httpNode, l)
        client.groups_filter[index] = make([]string, len(v.Filter))
        client.groups_filter[index] = append(client.groups_filter[index][:0], v.Filter...)
        log.Println("filter => ",client.groups_filter[index])
        for i := 0; i < l; i++ {
            client.groups[index][i] = &httpNode{
                url                : v.Nodes[i][0],
                send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
                send_times         : int64(0),
                send_failure_times : int64(0),
                is_down            : false,
                lock               : new(sync.Mutex),
                failure_times_flag : int32(0),
                cache_is_init      : false,
            }
        }
        index++
    }

    return client
}

func (client *HttpService) Start() {
    if !client.enable {
        return
    }
    go client.broadcast()
    for _, clients :=range client.groups {
        for _, h := range clients {
            go client.clientSendService(h)
        }
    }
}

// 初始化节点缓冲区，这个缓冲区用于存放发送失败的数据，最多HTTP_CACHE_LEN条
func (client *HttpService) cacheInit(node *httpNode) {
    if node.cache_is_init {
        return
    }

    log.Println("初始化cache")
    node.cache = make([][]byte, HTTP_CACHE_LEN)
    for k := 0; k < HTTP_CACHE_LEN; k++ {
        node.cache[k] = make([]byte, HTTP_CACHE_BUFFER_SIZE)
    }
    node.cache_is_init = true
    node.cache_index = 0
    node.cache_full = false
}

// 添加数据到缓冲区
func (client *HttpService) addCache(node *httpNode, msg []byte) {
    //log.Println("node cache ===> ", node.cache_index, cap(node.cache))
    node.cache[node.cache_index] = append(node.cache[node.cache_index][:0], msg...)
    node.cache_index++
    log.Println(node.cache_index, "添加cache数据")
    if node.cache_index >= HTTP_CACHE_LEN {
        node.cache_index = 0;
        node.cache_full = true
    }
}

// 尝试对失败的数据进行重发
func (client *HttpService) sendCache(node *httpNode) {
    if node.cache_index > 0 {
        //保持时序
        if node.cache_full {
            for j := node.cache_index; j < HTTP_CACHE_LEN; j++ {
                //重发
                log.Println(node.cache_index, "数据重发-full")
                node.send_queue <- node.cache[j]
            }
            node.cache_full = false
        }

        for j := 0; j < node.cache_index; j++ {
            //重发
            log.Println("数据重发")
            node.send_queue <- node.cache[j]
            node.cache_index--
        }
    }
}

// 节点故障检测与恢复服务
func (client *HttpService) errorCheckService(node *httpNode) {
    for {
        node.lock.Lock()
        if node.is_down {
            // 发送空包检测
            // post默认3秒超时，所以这里不会死锁
            _, err := http.Post(node.url,[]byte{byte(0)})
            if err == nil {
                //重新上线
                node.is_down = false
                log.Println(node.url, "节点恢复")
                //对失败的cache进行重发
                client.sendCache(node)
            }
        }
        node.lock.Unlock()
        time.Sleep(time.Second)
    }
}

// 节点服务协程
func (client *HttpService) clientSendService(node *httpNode) {
    go client.errorCheckService(node)

    to := time.NewTimer(time.Second*1)
    for {
        select {
        case  msg := <-node.send_queue:
            node.lock.Lock()
            if !node.is_down {
                atomic.AddInt64(&node.send_times, int64(1))
                log.Println("post到url：", node.url)
                data, err := http.Post(node.url, msg)

                if (err != nil) {
                    atomic.AddInt64(&client.send_failure_times, int64(1))
                    atomic.AddInt64(&node.send_failure_times, int64(1))
                    atomic.AddInt32(&node.failure_times_flag, int32(1))
                    failure_times := atomic.LoadInt32(&node.failure_times_flag)

                    // 如果连续3次错误，标志位故障
                    if failure_times >= 3 {
                        //发生故障
                        log.Println(node.url, "发生错误，下线节点")
                        node.is_down = true
                    }
                    log.Println(node.url, "失败次数：", node.send_failure_times)

                    client.cacheInit(node)
                    client.addCache(node, msg)

                } else {
                    if node.is_down {
                        node.is_down = false
                    }
                    failure_times := atomic.LoadInt32(&node.failure_times_flag)
                    //恢复即时清零故障计数
                    if failure_times > 0 {
                        atomic.StoreInt32(&node.failure_times_flag, 0)
                    }
                    //对失败的cache进行重发
                    client.sendCache(node)
                }
                log.Println(node.url, " post 返回值：", data)
            } else {
                // 故障节点，缓存需要发送的数据
                // 这里就需要一个map[string][10000][]byte，最多缓存10000条
                // 保持最新的10000条
                client.addCache(node, msg)
            }

            node.lock.Unlock()
        case <-to.C://time.After(time.Second*3):
        //log.Println("发送超时...", tcp)
        }
    }
}

// 广播服务
func (client *HttpService) broadcast() {
    to := time.NewTimer(time.Second*1)
    for {
        select {
        case  msg := <-client.send_queue:
            client.lock.Lock()
            for index, clients := range client.groups {
                // 如果分组里面没有客户端连接，跳过
                if len(clients) <= 0 {
                    continue
                }
                filter := client.groups_filter[index]
                flen   := len(filter)


                //2字节长度
                table_len := int(msg[0]) +
                    int(msg[1] << 8);
                table := string(msg[2:table_len+2])
                log.Println("事件发生的数据表：", table_len, table)
                //分组过滤
                //log.Println(filter)
                if flen > 0 {
                    is_match := false
                    for _, f := range filter {
                        match, err := regexp.MatchString(f, table)
                        if err != nil {
                            continue
                        }
                        if match {
                            is_match = true
                        }
                    }
                    if !is_match {
                        continue
                    }
                }

                for _, conn := range clients {
                    log.Println("http发送广播消息")
                    conn.send_queue <- msg[table_len+2:]
                }
            }
            client.lock.Unlock()
        case <-to.C://time.After(time.Second*3):
        }
    }
}

// 对外的广播发送接口
func (client *HttpService) SendAll(msg []byte) bool {
    if !client.enable {
        return false
    }
    if len(client.send_queue) >= cap(client.send_queue) {
        log.Println("http发送缓冲区满...")
        return false
    }
    client.send_queue <- msg
    return true
}
