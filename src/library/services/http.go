package services

import (
    "io/ioutil"
    "log"
    "errors"
    "bytes"
    "net"
    "time"
    "net/http"
    "sync/atomic"
    "sync"
    "strconv"
    "regexp"
)

const HTTP_POST_TIMEOUT = 3 //3秒超时
const HTTP_CACHE_LEN = 10000
const HTTP_CACHE_BUFFER_SIZE = 4096

var ERR_STATUS error =  errors.New("错误的状态码")

type HttpService struct {
    send_queue chan []byte     // 发送channel
    groups [][]*httpNode       // 客户端分组，现在支持两种分组，广播组合负载均衡组
    groups_mode []int          // 分组的模式 1，2 广播还是复载均衡
    groups_filter [][]string     // 分组过滤器
    lock *sync.Mutex           // 互斥锁，修改资源时锁定
    send_failure_times int64   // 发送失败次数
    enable bool
}

type httpNode struct {
    url string                  // url
    send_queue chan []byte      // 发送channel
    weight int                  // 权重 0 - 100
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
        groups_mode        : make([]int, glen),
        groups_filter      : make([][]string, glen),
        send_failure_times : int64(0),
        enable             : config.Enable,
    }
    index := 0
    for _, v := range config.Groups {
        l := len(v.Nodes)
        client.groups[index]      = make([]*httpNode, l)
        client.groups_mode[index] = v.Mode
        client.groups_filter[index] = make([]string, len(v.Filter))
        client.groups_filter[index] = append(client.groups_filter[index][:0], v.Filter...)
        log.Println("filter => ",client.groups_filter[index])
        for i := 0; i < l; i++ {
            w, _ := strconv.Atoi(v.Nodes[i][1])
            client.groups[index][i] = &httpNode{
                url                : v.Nodes[i][0],
                weight             : w,
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
            _, err := client.post(node.url,[]byte{byte(0)})
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
                data, err := client.post(node.url, msg)

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
                // 分组的模式
                mode   := client.groups_mode[index]
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

                // 如果不等于权重，即广播模式
                if mode != MODEL_WEIGHT {
                    for _, conn := range clients {
                        log.Println("http发送广播消息")
                        conn.send_queue <- msg[table_len+2:]
                    }
                } else {
                    // 负载均衡模式
                    // todo 根据已经send_times的次数负载均衡
                    clen := len(clients)
                    target := clients[0]
                    //将发送次数/权重 作为负载基数，每次选择最小的发送
                    js := float64(atomic.LoadInt64(&target.send_times))/float64(target.weight)
                    for i := 1; i < clen; i++ {
                        stimes := atomic.LoadInt64(&clients[i].send_times)
                        //conn.send_queue <- msg
                        if stimes == 0 {
                            //优先发送没有发过的
                            target = clients[i]
                            break
                        }
                        _js := float64(stimes)/float64(clients[i].weight)
                        log.Println("权重基数",float64(stimes),float64(clients[i].weight), _js, js)
                        if _js < js {
                            js = _js
                            target = clients[i]
                        }
                    }
                    log.Println("http发送权重消息，", (*target).url)
                    target.send_queue <- msg[table_len+2:]
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

func (client *HttpService) post(addr string, post_data []byte) ([]byte, error) {
    req, err := http.NewRequest("POST", addr, bytes.NewReader(post_data))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Connection", "close")
    // 超时设置
    DefaultClient := http.Client{
        Transport: &http.Transport {
            Dial: func(netw, addr string) (net.Conn, error) {
                deadline := time.Now().Add(HTTP_POST_TIMEOUT * time.Second)
                c, err := net.DialTimeout(netw, addr, time.Second * HTTP_POST_TIMEOUT)
                if err != nil {
                    return nil, err
                }
                c.SetDeadline(deadline)
                return c, nil
            },
        },
    }
    // 执行post
    resp, err := DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    // 关闭io
    defer resp.Body.Close()
    // 判断返回状态
    if resp.StatusCode != http.StatusOK {
        // 返回异常状态
        log.Println("http post error, error status back: ", resp.StatusCode)
        return nil, ERR_STATUS
    }
    data, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    return data, nil
}

func (client *HttpService) get(addr string) (*[]byte, *int, *http.Header, error) {
    // 上传JSON数据
    req, e := http.NewRequest("GET", addr, nil)
    if e != nil {
        // 返回异常
        return nil, nil, nil, e
    }
    // 完成后断开连接
    req.Header.Set("Connection", "close")
    // -------------------------------------------
    // 设置 TimeOut
    DefaultClient := http.Client{
        Transport: &http.Transport{
            Dial: func(netw, addr string) (net.Conn, error) {
                deadline := time.Now().Add(30 * time.Second)
                c, err := net.DialTimeout(netw, addr, time.Second*30)
                if err != nil {
                    return nil, err
                }
                c.SetDeadline(deadline)
                return c, nil
            },
        },
    }
    // -------------------------------------------
    // 执行
    resp, ee := DefaultClient.Do(req)
    if ee != nil {
        // 返回异常
        return nil, nil, nil, ee
    }
    // 保证I/O正常关闭
    defer resp.Body.Close()
    // 判断请求状态
    if resp.StatusCode == 200 {
        data, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            // 读取错误,返回异常
            return nil, nil, nil, err
        }
        // 成功，返回数据及状态
        return &data, &resp.StatusCode, &resp.Header, nil
    } else {
        // 失败，返回状态
        return nil, &resp.StatusCode, nil, nil
    }
    // 不会到这里
    return nil, nil, nil, nil
}
