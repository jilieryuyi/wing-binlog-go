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
	_ "flag"
	log "github.com/sirupsen/logrus"
	_ "library/cluster"
	"library/unix"
	"library/command"
	"flag"
	"time"
	"context"
)

var (
	debug = flag.Bool("debug", false, "启用调试模式，默认为false")
	version = flag.Bool("version", false, "版本信息")
	stop = flag.Bool("stop", false, "停止服务")
	//-service-reload http
	//-service-reload tcp
	//-service-reload websocket
	//-service-reload kafka
	//-service-reload all ##重新加载全部服务
	service_reload = flag.String("service-reload", "", "重新加载配置，比如重新加载http服务配置：-service-reload http")
)

const (
	VERSION = "1.0.0"
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
	//syslog.Println("debug",*debug)
	//if len(os.Args) > 1 && os.Args[1] == "stop" {
	//	killPid()
	//	return
	//}
	if (*version) {
		fmt.Println(VERSION)
		return
	}
	if (*stop) {
		command.Stop()
		return
	}
	if *service_reload != "" {
		command.Reload(*service_reload)
		return
	}
 	pprofService()
	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu) //指定cpu为多核运行 旧版本兼容
	ctx, cancel := context.WithCancel(context.Background())

	tcp_service       := services.NewTcpService()
	websocket_service := services.NewWebSocketService()
	http_service      := services.NewHttpService()
	kafaka_service    := services.NewKafkaService()

	blog := binlog.NewBinlog()
	// 注册服务
	blog.BinlogHandler.RegisterService("tcp", tcp_service)
	blog.BinlogHandler.RegisterService("websocket", websocket_service)
	blog.BinlogHandler.RegisterService("http", http_service)
	blog.BinlogHandler.RegisterService("kafka", kafaka_service)
	blog.Start(&ctx)

	server := unix.NewUnixServer()
	server.Start(blog, &cancel)

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
	fmt.Println("程序退出...")
	//time.Sleep(time.Second)
}
