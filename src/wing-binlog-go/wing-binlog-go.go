package main

import (
	"library/binlog"
	"library/services"
	"library/app"
	_ "github.com/go-sql-driver/mysql"
	"runtime"
	"os"
	"os/signal"
	"syscall"
	_ "net/http/pprof"
	"net/http"
	"fmt"
	"io/ioutil"
	"strconv"
	"library/file"
	"flag"
	log "github.com/sirupsen/logrus"
	"library/unix"
	"library/command"
	"time"
	"context"
)

var (
	debug = flag.Bool("debug", false, "enable debug, default disable")
	version = flag.Bool("version", false, "wing binlog go version")
	stop = flag.Bool("stop", false, "stop service")
	//-service-reload http
	//-service-reload tcp
	//-service-reload websocket
	//-service-reload kafka ##暂时去掉了，暂时不支持kafka
	//-service-reload all ##重新加载全部服务
	service_reload = flag.String("service-reload", "", "reload service config, usage: -service-reload  all|http|tcp|websocket")
	help = flag.Bool("help", false, "help")
	joinTo = flag.String("join-to", "", "join to cluster")
)

const (
	VERSION = "1.0.0"
)

var pid = file.GetCurrentPath() + "/wing-binlog-go.pid"
func writePid() {
	var data_str = []byte(fmt.Sprintf("%d", os.Getpid()));
	ioutil.WriteFile(pid, data_str, 0777)  //写入文件(字节数组)
}

func clearPid() {
	f := file.WFile{pid}
	f.Delete()
}

func killPid() {
	dat, _ := ioutil.ReadFile(pid)
	fmt.Print(string(dat))
	pid, _ := strconv.Atoi(string(dat))
	log.Println("给进程发送终止信号：", pid)
	//err := syscall.Kill(pid, syscall.SIGTERM)
	//log.Println(err)
}

func pprofService() {
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

		//下载文件 http://localhost:6060/debug/pprof/profile
		//分析 go tool pprof -web /Users/yuyi/Downloads/profile
		log.Println(http.ListenAndServe("0.0.0.0:6060", nil))
	}()
}

func usage() {
	fmt.Println("*********************************************************************")
	fmt.Println("wing-binlog-go                             --start service")
	fmt.Println("wing-binlog-go -version                    --show version info")
	fmt.Println("wing-binlog-go -stop                       --stop service")
	fmt.Println("wing-binlog-go -service-reload  http       --reload http service")
	fmt.Println("wing-binlog-go -service-reload  tcp        --reload tcp service")
	fmt.Println("wing-binlog-go -service-reload  websocket  --reload websocket service")
	fmt.Println("wing-binlog-go -service-reload  all        --reload all service")
	fmt.Println("*********************************************************************")
}

func init() {
	time.LoadLocation("Local")
	log.SetFormatter(&log.TextFormatter{TimestampFormat:"2006-01-02 15:04:05",
		ForceColors:true,
		QuoteEmptyFields:true, FullTimestamp:true})
	app_config, _ := app.GetAppConfig()
	log.SetLevel(log.Level(app_config.LogLevel))//log.DebugLevel)
	//log.Debugf("wing-binlog-go基础配置：%+v\n", app_config)
	//log.ResetOutHandler()
	//u := data.User{"admin", "admin"}
	//log.Println("用户查询：",u.Get())
	//u = data.User{"admin", "admin1"}
	//log.Println("用户查询：",u.Get())
	writePid()
	//标准输出重定向
	//library.Reset()
}

func main() {
	flag.Parse()
	// 显示版本信息
	if (*version) {
		fmt.Println(VERSION)
		return
	}
	// 停止服务
	if (*stop) {
		command.Stop()
		return
	}
	// 重新加载服务
	if *service_reload != "" {
		command.Reload(*service_reload)
		return
	}
	// 帮助
	if *help {
		usage()
		return
	}
	// 加入集群
	if *joinTo != "" {
		command.JoinTo(*joinTo)
		return
	}

	// 退出程序时删除pid文件
	defer clearPid()
	// 性能测试
 	pprofService()
	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu) //指定cpu为多核运行 旧版本兼容
	ctx, cancel := context.WithCancel(context.Background())

	// 各种通信服务
	tcp_service       := services.NewTcpService()
	websocket_service := services.NewWebSocketService()
	http_service      := services.NewHttpService()

	// 核心binlog服务
	blog := binlog.NewBinlog(&ctx)
	// 注册tcp、http、websocket服务
	blog.BinlogHandler.RegisterService("tcp", tcp_service)
	blog.BinlogHandler.RegisterService("websocket", websocket_service)
	blog.BinlogHandler.RegisterService("http", http_service)
	blog.Start()

	// unix socket服务，用户本地指令控制
	server := unix.NewUnixServer()
	server.Start(blog, &cancel, pid)
	defer server.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sc
	// 优雅的退出程序
	cancel()
	blog.Close()
	fmt.Println("服务退出...")
}
