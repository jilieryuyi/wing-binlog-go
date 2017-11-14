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
)

const HTTP_POST_TIMEOUT = 3 //3秒超时
var ERR_STATUS error =  errors.New("错误的状态码")

type HttpService struct {
    send_queue chan []byte     // 发送channel
    groups [][]*httpNode       // 客户端分组，现在支持两种分组，广播组合负载均衡组
    groups_mode []int          // 分组的模式 1，2 广播还是复载均衡
    lock *sync.Mutex           // 互斥锁，修改资源时锁定
    send_failure_times int64   // 发送失败次数
}

type httpNode struct {
    url string                  // url
    send_queue chan []byte      // 发送channel
    weight int                  // 权重 0 - 100
    send_times int64            // 发送次数
    send_failure_times int64    // 发送失败次数
}

func NewHttpService(config *HttpConfig) *HttpService {
    glen := len(config.Groups)
    client := &HttpService {
        send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
        lock               : new(sync.Mutex),
        groups             : make([][]*httpNode, glen),
        groups_mode        : make([]int, glen),
        send_failure_times : int64(0),
    }
    index := 0
    for _, v := range config.Groups {
        l := len(v.Nodes)
        client.groups[index]      = make([]*httpNode, l)
        client.groups_mode[index] = v.Mode

        for i := 0; i < l; i++ {
            w, _ := strconv.Atoi(v.Nodes[i][1])
            client.groups[index][i] = &httpNode{
                url                : v.Nodes[i][0],
                weight             : w,
                send_queue         : make(chan []byte, TCP_MAX_SEND_QUEUE),
                send_times         : int64(0),
                send_failure_times : int64(0),
            }
        }
        index++
    }

    return client
}

func (client *HttpService) Start() {
    go client.broadcast()
    for _, clients :=range client.groups {
        for _, h := range clients {
            go client.clientSendService(h)
        }
    }
}

func (client *HttpService) clientSendService(node *httpNode) {
    to := time.NewTimer(time.Second*1)
    for {
        select {
        case  msg := <-node.send_queue:
            atomic.AddInt64(&node.send_times, int64(1))
            log.Println("post到url：",node.url)
            data, err := client.post(node.url, msg)
            if (err != nil) {
                atomic.AddInt64(&client.send_failure_times, int64(1))
                atomic.AddInt64(&node.send_failure_times, int64(1))
                log.Println(node.url, "失败次数：", node.send_failure_times)
            }
            log.Println(node.url, " post 返回值：", data)
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
                mode := client.groups_mode[index]
                // 如果不等于权重，即广播模式
                if mode != MODEL_WEIGHT {
                    for _, conn := range clients {
                        log.Println("http发送广播消息")
                        conn.send_queue <- msg
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
                    target.send_queue <- msg
                }
            }
            client.lock.Unlock()
        case <-to.C://time.After(time.Second*3):
        }
    }
}

// 对外的广播发送接口
func (client *HttpService) SendAll(msg []byte) bool {
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
