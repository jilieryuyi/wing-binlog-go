package http

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"library/data"
	"library/path"
	"net/http"
	"sync"
	"time"
	wstring "library/string"
)

var online_users map[string]*OnLineUser = make(map[string]*OnLineUser)
var online_users_lock *sync.Mutex = new(sync.Mutex)
var http_errors map[int]string = map[int]string{
	200: "ok",
	201: "login error",
	202: "logout error",
	203: "logout ok",
	204: "please relogin",
}

// 初始化，系统自动执行
func init() {
	return
	log.SetFormatter(&log.TextFormatter{TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true, FullTimestamp: true,
	})
	config, _ := getServiceConfig()
	ws := NewWebSocketService(config.Websocket.Listen, config.Websocket.Port)
	ws.Start()
	server := &HttpServer{
		Path:        path.CurrentPath + "/web",
		Ip:          config.Http.Listen,
		Port:        config.Http.Port,
		ws:          ws,
		httpHandler: http.FileServer(http.Dir(path.CurrentPath + "/web")),
	}
	server.Start()
}

// 判断签名是否在线
func isOnline(sign string) bool {
	online_users_lock.Lock()
	user, ok := online_users[sign]
	if ok {
		user.LastPostTime = time.Now().Unix()
	}
	online_users_lock.Unlock()
	return ok
}

func output(code int, msg string, data string) string {
	return fmt.Sprintf("{\"code\":%d, \"message\":\"%s\", \"data\":\"%s\"}", code, msg, data)
}

func onUserLogin(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	username, ok1 := req.Form["username"]
	password, ok2 := req.Form["password"]
	log.Println("http服务", req.Form, username[0], password[0])
	user := data.User{username[0], password[0]}
	if ok1 && ok2 && user.Get() {
		log.Infof("http服务，用户 %s 登陆成功", username[0])
		user_sign := wstring.RandString(32)
		online_users_lock.Lock()
		online_users[user_sign] = &OnLineUser{
			Name:         username[0],
			Password:     password[0],
			LastPostTime: time.Now().Unix(),
		}
		online_users_lock.Unlock()
		cookie := http.Cookie{
			Name:   "user_sign",
			Value:  user_sign,
			Path:   "/",
			MaxAge: 86400,
		}
		http.SetCookie(w, &cookie)
		w.Write([]byte(output(200, http_errors[200], "")))
	} else {
		w.Write([]byte(output(201, http_errors[201], "")))
	}
}

func onUserLogout(w http.ResponseWriter, req *http.Request) {
	user_sign, err := req.Cookie("user_sign")
	if err != nil {
		w.Write([]byte(output(202, http_errors[202], "")))
		return
	}
	cookie := http.Cookie{
		Name:   "user_sign",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, &cookie)
	online_users_lock.Lock()
	_, ok := online_users[user_sign.Value]
	if ok {
		log.Infof("http服务删除在线用户：%s", user_sign.Value)
		delete(online_users, user_sign.Value)
		w.Write([]byte(output(203, http_errors[203], "")))
	} else {
		log.Infof("http服务删除在线用户错误，session不存在：%s", user_sign.Value)
		w.Write([]byte(output(202, http_errors[202], "")))
	}
	online_users_lock.Unlock()
}

func (server *HttpServer) checkLoginTimeout() {
	for {
		online_users_lock.Lock()
		for key, user := range online_users {
			now := time.Now().Unix()
			if now-user.LastPostTime > DEFAULT_LOGIN_TIMEOUT {
				log.Infof("http服务检测到登录超时：", key, user.Name)
				server.ws.DeleteClient(key)
				delete(online_users, key)
			}
		}
		online_users_lock.Unlock()
		time.Sleep(time.Second * 3)
	}
}

func (server *HttpServer) Start() {
	go server.checkLoginTimeout()
	go func() {
		log.Infof("http服务器启动...")
		log.Infof("http服务监听: %s:%d", server.Ip, server.Port)
		static_http_handler := http.FileServer(http.Dir(server.Path))
		http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			// 判断是否在线
			user_sign, err := req.Cookie("user_sign")
			is_leave := false
			if err != nil {
				is_leave = true
			} else {
				online_users_lock.Lock()
				user, ok := online_users[user_sign.Value]
				if !ok {
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
					Name:   "user_sign",
					Path:   "/",
					MaxAge: -1,
				}
				http.SetCookie(w, &cookie)
			}
			static_http_handler.ServeHTTP(w, req)
		})
		http.HandleFunc("/user/login", onUserLogin)
		http.HandleFunc("/get/websocket/port", func(w http.ResponseWriter, req *http.Request) {
			user_sign, err := req.Cookie("user_sign")
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
		http.HandleFunc("/user/logout", func(w http.ResponseWriter, req *http.Request) {
			user_sign, err := req.Cookie("user_sign")
			if err == nil {
				server.ws.DeleteClient(user_sign.Value)
			}
			onUserLogout(w, req)
		})
		dns := fmt.Sprintf("%s:%d", server.Ip, server.Port)
		err := http.ListenAndServe(dns, nil)
		if err != nil {
			log.Fatalf("http服务启动失败: %v", err)
		}
	}()
}
