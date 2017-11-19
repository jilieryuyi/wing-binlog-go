package http

import (
    "net/http"
    "bytes"
    "net"
    "time"
    "io/ioutil"
    "log"
    "errors"
   // "io"
    "fmt"
    "path/filepath"
    "os"
    "strings"
   // "io"
    "library/admin"
)

type HttpServer struct{
    Path string // web路径 当前路径/web
    Ip string   // 监听ip 0.0.0.0
    Port int    // 9989
}

const HTTP_POST_TIMEOUT = 3 //3秒超时
var ERR_STATUS error =  errors.New("错误的状态码")

func Post(addr string, post_data []byte) ([]byte, error) {
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
func Get(addr string) (*[]byte, *int, *http.Header, error) {
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

func (server *HttpServer) Start() {
    go func() {
        log.Println("http服务器启动...")
        log.Printf("http监听: %s:%d", server.Ip, server.Port)

        staticHandler := http.FileServer(http.Dir(server.Path))
        http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
            log.Println("http请求: ",w, req.URL.Path)
            //if req.URL.Path == "/" {
                staticHandler.ServeHTTP(w, req)
            //    return
            //}
           // io.WriteString(w, "hello, world!\n")
        })
        dns := fmt.Sprintf("%s:%d", server.Ip, server.Port)
        err := http.ListenAndServe(dns, nil)
        if err != nil {
            log.Fatal("启动http服务失败: ", err)
        }
    }()
}

func init() {
    ws := admin.NewWebSocketService("0.0.0.0", 9988);
    ws.Start()

    dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
        log.Fatal(err)
    }
    path := strings.Replace(dir, "\\", "/", -1)
    server := &HttpServer{
        Path: path+"/web",
        Ip:"0.0.0.0",
        Port:9989,
    }
    server.Start()
}
