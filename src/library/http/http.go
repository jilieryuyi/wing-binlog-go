package http

import (
    "net/http"
    "bytes"
    "net"
    "time"
    "io/ioutil"
    log "github.com/sirupsen/logrus"
    "errors"
    "fmt"
    "path/filepath"
    "os"
    "strings"
    "math/rand"
    "library/data"
    "sync"
)

type HttpServer struct{
    Path string // web路径 当前路径/web
    Ip string   // 监听ip 0.0.0.0
    Port int    // 9989
    ws *WebSocketService
}

type OnLineUser struct {
    Name string
    Password string
    LastPostTime int64
}

var online_users map[string] *OnLineUser = make(map[string] *OnLineUser)
var online_users_lock *sync.Mutex = new(sync.Mutex)


func is_online(sign string) bool {
    online_users_lock.Lock()
    _, ok := online_users[sign]
    online_users_lock.Unlock()
    return ok
}


var http_errors map[int] string = map[int] string{
    200 : "login ok",
    201 : "login error",
    202 : "logout error",
    203 : "logout ok",
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
    res, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }
    return res, nil
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
        res, err := ioutil.ReadAll(resp.Body)
        if err != nil {
            // 读取错误,返回异常
            return nil, nil, nil, err
        }
        // 成功，返回数据及状态
        return &res, &resp.StatusCode, &resp.Header, nil
    } else {
        // 失败，返回状态
        return nil, &resp.StatusCode, nil, nil
    }
    // 不会到这里
    return nil, nil, nil, nil
}

func init() {
    ws := NewWebSocketService("0.0.0.0", 9988);
    ws.Start()

    dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
    if err != nil {
        log.Fatal(err)
    }
    path := strings.Replace(dir, "\\", "/", -1)
    server := &HttpServer{
        Path : path+"/web",
        Ip   : "0.0.0.0",
        Port : 9989,
        ws   : ws,
    }
    server.Start()
    //ws.http = server
}

func randString() string {
    str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
    bt := []byte(str)
    result := []byte{}
    r := rand.New(rand.NewSource(time.Now().UnixNano()))
    for i := 0; i < 32; i++ {
        result = append(result, bt[r.Intn(len(bt))])
    }
    return string(result)
}

func output(code int, msg string) string{
    return fmt.Sprintf("{\"code\":%d, \"message\":\"%s\"}", code, msg)
}

func OnUserLogin(w http.ResponseWriter, req *http.Request) {
    req.ParseForm()
    username, ok1 := req.Form["username"]
    password, ok2 := req.Form["password"]
    log.Println(req.Form, username[0], password[0])
    user := data.User{username[0], password[0]}
    if ok1 && ok2 && user.Get() {
        log.Println("login success")
        user_sign := randString()
        online_users_lock.Lock()
        online_users[user_sign] = &OnLineUser{
            Name : username[0],
            Password:password[0],
            LastPostTime:time.Now().Unix(),
        }
        online_users_lock.Unlock()
        cookie := http.Cookie{
            Name: "user_sign",
            Value: user_sign,
            Path: "/",
            MaxAge: 86400,
        }
        http.SetCookie(w, &cookie)
        w.Write([]byte(output(200, http_errors[200])))
    } else {
        w.Write([]byte(output(201, http_errors[201])))
    }
}

func OnUserLogout(w http.ResponseWriter, req *http.Request) {
    user_sign, err:= req.Cookie("user_sign")
    if err != nil {
        w.Write([]byte(output(202, http_errors[202])))
        return
    }
    cookie := http.Cookie{
        Name: "user_sign",
        Path: "/",
        MaxAge: -1,
    }
    http.SetCookie(w, &cookie)

    online_users_lock.Lock()
    _, ok := online_users[user_sign.Value]
    if ok {
        log.Println("delete online user ", user_sign.Value)
        delete(online_users, user_sign.Value)
        w.Write([]byte(output(203, http_errors[203])))
    } else {
        log.Println("delete error, session does not exists ", user_sign.Value)
        w.Write([]byte(output(202, http_errors[202])))
    }
    online_users_lock.Unlock()
}

func (server *HttpServer) Start() {
    go func() {
        //登录超时检测
        for {
            online_users_lock.Lock()
            for key, user := range online_users {
                now := time.Now().Unix()
                // 10分钟不活动强制退出
                if now - user.LastPostTime > 600 {
                    server.ws.DeleteClient(key)
                    delete(online_users, key)
                    log.Println("登录超时...", key)
                }
            }
            online_users_lock.Unlock()
            time.Sleep(time.Second*3)
        }
    }()
    go func() {
        log.Println("http服务器启动...")
        log.Printf("http监听: %s:%d", server.Ip, server.Port)
        static_http_handler := http.FileServer(http.Dir(server.Path))

        http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request){
            // 判断是否在线
            user_sign, err := req.Cookie("user_sign")
            is_leave := false
            if err != nil {
                is_leave = true
            } else {
                _, ok := online_users[user_sign.Value]
                if !ok {
                    // 清除ws连接
                    //server.ws.DeleteClient(user_sign.Value)
                    is_leave = true
                }
            }
            if is_leave {
                // 清除cookie
                cookie := http.Cookie{
                    Name: "user_sign",
                    Path: "/",
                    MaxAge: -1,
                }
                http.SetCookie(w, &cookie)
            }
            static_http_handler.ServeHTTP(w, req)
        })
        http.HandleFunc("/user/login", OnUserLogin)
        http.HandleFunc("/user/logout", func(w http.ResponseWriter, req *http.Request){
            user_sign, err:= req.Cookie("user_sign")
            if err == nil {
                server.ws.DeleteClient(user_sign.Value)
            }
            OnUserLogout(w, req)
        })

        dns := fmt.Sprintf("%s:%d", server.Ip, server.Port)
        err := http.ListenAndServe(dns, nil)
        if err != nil {
            log.Fatal("启动http服务失败: ", err)
        }
    }()
}

