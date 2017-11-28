package http

import (
    "net/http"
    "time"
    log "github.com/sirupsen/logrus"
    "errors"
    "fmt"
    "math/rand"
    "library/data"
    "library/file"
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
    user, ok := online_users[sign]
    if ok {
        user.LastPostTime = time.Now().Unix()
    }
    online_users_lock.Unlock()
    return ok
}


var http_errors map[int] string = map[int] string{
    200 : "ok",
    201 : "login error",
    202 : "logout error",
    203 : "logout ok",
    204 : "please relogin",
}

const HTTP_POST_TIMEOUT = 3 //3秒超时
var ERR_STATUS error =  errors.New("错误的状态码")


func init() {
    config, _ := getServiceConfig()
    ws := NewWebSocketService(config.Websocket.Listen, config.Websocket.Port);
    ws.Start()

    //dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
    //if err != nil {
    //    log.Fatal(err)
    //}
    //path := strings.Replace(dir, "\\", "/", -1)
    server := &HttpServer{
        Path : file.GetCurrentPath()+"/web",
        Ip   : config.Http.Listen,
        Port : config.Http.Port,
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

func output(code int, msg string, data string) string{
    return fmt.Sprintf("{\"code\":%d, \"message\":\"%s\", \"data\":\"%s\"}", code, msg, data)
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
        w.Write([]byte(output(200, http_errors[200], "")))
    } else {
        w.Write([]byte(output(201, http_errors[201], "")))
    }
}

func OnUserLogout(w http.ResponseWriter, req *http.Request) {
    user_sign, err:= req.Cookie("user_sign")
    if err != nil {
        w.Write([]byte(output(202, http_errors[202], "")))
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
        w.Write([]byte(output(203, http_errors[203], "")))
    } else {
        log.Println("delete error, session does not exists ", user_sign.Value)
        w.Write([]byte(output(202, http_errors[202], "")))
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
                online_users_lock.Lock()
                user, ok := online_users[user_sign.Value]
                if !ok {
                    // 清除ws连接
                    //server.ws.DeleteClient(user_sign.Value)
                    is_leave = true
                } else {
                    // 跟新空闲时间
                    user.LastPostTime = time.Now().Unix()
                }
                online_users_lock.Unlock()
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
        http.HandleFunc("/get/websocket/port", func(w http.ResponseWriter, req *http.Request) {
            user_sign, err:= req.Cookie("user_sign")
            if err != nil {
                w.Write([]byte(output(204, http_errors[204], "")))
                return
            }

            online_users_lock.Lock()
            _, ok := online_users[user_sign.Value]
            online_users_lock.Unlock()
            if ok {
                w.Write([]byte(output(200, http_errors[200], fmt.Sprintf("%d", server.ws.Port))))
            } else {
                w.Write([]byte(output(204, http_errors[204], "")))
            }
        })

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

