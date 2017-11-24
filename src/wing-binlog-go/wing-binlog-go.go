package main

import (
	log "library/log"
	"library"
	"library/services"
	_ "library/data"
	_ "github.com/go-sql-driver/mysql"
	"runtime"
	"os"
	"os/signal"
	"syscall"
	_"net/http/pprof"
	"net/http"
	"fmt"
	"io/ioutil"
	"strconv"
	"library/data"
	"library/file"
	"flag"
	syslog "log"
	"github.com/sirupsen/logrus"
)


func writePid() {
	var data_str = []byte(fmt.Sprintf("%d", os.Getpid()));
	ioutil.WriteFile(file.GetCurrentPath() + "/wing-binlog-go.pid", data_str, 0777)  //写入文件(字节数组)
}

func killPid() {
	dat, _ := ioutil.ReadFile(file.GetCurrentPath() + "/wing-binlog-go.pid")
	fmt.Print(string(dat))
	pid, _ := strconv.Atoi(string(dat))
	log.Println("给进程发送终止信号：", pid)
	//err := syscall.Kill(pid, syscall.SIGTERM)
	//log.Println(err)
}

func init() {
    fmt.Println("log init------------")
    logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat:"2006-01-02 15:04:05",
        ForceColors:true,
        QuoteEmptyFields:true, FullTimestamp:true})
    log.ResetOutHandler()
}

var debug = flag.Bool("debug", false, "enable debug, default true")

func main() {
	flag.Parse()
	syslog.Println("debug", *debug)
	log.ResetOutHandler()
	u := data.User{"admin", "admin"}
	log.Println("用户查询：",u.Get())

	u = data.User{"admin", "admin1"}
	log.Println("用户查询：",u.Get())

	if len(os.Args) > 1 && os.Args[1] == "stop" {
		killPid()
		return
	}

	writePid()

	//标准输出重定向
	//library.Reset()
	go func() {
		//http://localhost:6060/debug/pprof/  内存性能分析工具
		//go tool pprof logDemo.exe --text a.prof
		//go tool pprof your-executable-name profile-filename
		//go tool pprof your-executable-name http://localhost:6060/debug/pprof/heap
		//go tool pprof wing-binlog-go http://localhost:6060/debug/pprof/heap
		//https://lrita.github.io/2017/05/26/golang-memory-pprof/
		//然后执行 text
		//go tool pprof -alloc_space http://127.0.0.1:6060/debug/pprof/heap
		//top20 -cum
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	//log.SetFlags(log.Ldate|log.Ltime|log.Lshortfile)
	//wing_log := library.GetLogInstance()
	//释放日志资源
	//defer library.FreeLogInstance()

	/*file := &library.WFile{"C:\\__test.txt"}
	str := file.ReadAll()
	//if str != "123" {
		log.Println("ReadAll error: ==>" + str + "<==", len(str))
	//}
	return*/

	//log.SetFlags(log.LstdFlags | log.Lshortfile)


	cpu := runtime.NumCPU()
	//指定cpu为多核运行 旧版本兼容
	runtime.GOMAXPROCS(cpu)

	current_path := file.GetCurrentPath()
	log.Println(current_path)

	config_file := current_path + "/config/mysql.toml"
	tcp_config_file := current_path + "/config/tcp.toml"
	websocket_config_file := current_path + "/config/websocket.toml"
	http_config_file := current_path + "/config/http.toml"

	//config_obj := &library.Ini{config_file}
	//config := config_obj.Parse()
	//if config == nil {
	//	log.Println("read config file: " + config_file + " error")
	//	return
	//}
	//log.Println(config)


	//config_file := "/tmp/__test_mysql.toml"
	config := &library.WConfig{config_file}

	app_config, err:= config.GetMysql()

	if err != nil {
		log.Println(err)
		return
	}


	// tcp服务
	wtcp_config := &library.WConfig{tcp_config_file}
	//{map[1:{1 group1} 2:{2 group2}] {0.0.0.0 9998}}
	tcp_config, err := wtcp_config.GetTcp()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("tcp config: ", tcp_config)
	tcp_service := services.NewTcpService(tcp_config)
	tcp_service.Start()

	// websocket 服务
	wwebsocket_config := &library.WConfig{websocket_config_file}
	//{map[1:{1 group1} 2:{2 group2}] {0.0.0.0 9998}}
	websocket_config, err := wwebsocket_config.GetTcp()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("websocket config: ",websocket_config)
	websocket_service := services.NewWebSocketService(websocket_config)
	websocket_service.Start()

	// http服务
	whttp_config := &library.WConfig{http_config_file}
	//{map[1:{1 group1} 2:{2 group2}] {0.0.0.0 9998}}
	http_config, err := whttp_config.GetHttp()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println("http config: ",http_config)
	http_service := services.NewHttpService(http_config)
	http_service.Start()

	blog := library.Binlog{DB_Config:app_config}
	defer blog.Close()
	blog.Start(tcp_service, websocket_service, http_service)

	<-sc

	//redis := &subscribe.Redis{}
	//tcp := &subscribe.Tcp{}
    //
	////subscribes
	//notify := []base.Subscribe{redis, tcp}
	//binlog := &workers.Binlog{}
    //
	//defer binlog.End(notify)
    //
	//binlog.Start(notify)
	//binlog.Loop(notify)
}
