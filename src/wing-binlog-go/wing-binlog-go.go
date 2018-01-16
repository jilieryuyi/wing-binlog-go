package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"
	"library/app"
	"library/binlog"
	"library/command"
	"library/file"
	"library/services"
	"library/unix"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

var (
	//if debug is true, print stack log
	debug   = flag.Bool("debug", false, "enable debug, default disable")
	version = flag.Bool("version", false, "wing binlog go version")
	stop    = flag.Bool("stop", false, "stop service")
	//-service-reload http
	//-service-reload tcp
	//-service-reload websocket
	//-service-reload kafka ##暂时去掉了，暂时不支持kafka
	//-service-reload all ##重新加载全部服务
	service_reload = flag.String("service-reload", "", "reload service config, usage: -service-reload  all|http|tcp|websocket")
	help           = flag.Bool("help", false, "help")
	joinTo         = flag.String("join-to", "", "join to cluster")
	members        = flag.Bool("members", false, "show members from current node")
)

const (
	VERSION = "1.0.0"
)

var (
	pid = file.GetCurrentPath() + "/wing-binlog-go.pid"
 	appConfig, _ = app.GetAppConfig()
)
// write pid file
func writePid() {
	data := []byte(fmt.Sprintf("%d", os.Getpid()))
	ioutil.WriteFile(pid, data, 0777)
}
// delete pid file
func clearPid() {
	f := file.WFile{pid}
	f.Delete()
}
// kill process by pid file
func killPid() {
	dat, _ := ioutil.ReadFile(pid)
	fmt.Print(string(dat))
	pid, _ := strconv.Atoi(string(dat))
	log.Debugf("try to kill process: %d", pid)
	//err := syscall.Kill(pid, syscall.SIGTERM)
	//log.Println(err)
}
// pprof tool support
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
		if appConfig.PprofListen != "" {
			http.ListenAndServe(appConfig.PprofListen, nil)
		}
	}()
}

func usage() {
	fmt.Println("*********************************************************************")
	fmt.Println("wing-binlog-go                                   --start service")
	fmt.Println("wing-binlog-go -version                          --show version info")
	fmt.Println("wing-binlog-go -stop                             --stop service")
	fmt.Println("wing-binlog-go -service-reload  http             --reload http service")
	fmt.Println("wing-binlog-go -service-reload  tcp              --reload tcp service")
	fmt.Println("wing-binlog-go -service-reload  websocket        --reload websocket service")
	fmt.Println("wing-binlog-go -service-reload  all              --reload all service")
	fmt.Println("wing-binlog-go -join-to  [serviceIp:servicePort] --join to cluster")
	fmt.Println("wing-binlog-go -members                          --show cluster members")
	fmt.Println("*********************************************************************")
}

func init() {
	time.LoadLocation(appConfig.TimeZone)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.SetLevel(log.Level(appConfig.LogLevel)) //log.DebugLevel)
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

func commandService() bool {
	// 显示版本信息
	if *version {
		fmt.Println(VERSION)
		return true
	}
	// 停止服务
	if *stop {
		command.Stop()
		return true
	}
	// 重新加载服务
	if *service_reload != "" {
		command.Reload(*service_reload)
		return true
	}
	// 帮助
	if *help {
		usage()
		return true
	}
	// 加入集群
	if *joinTo != "" {
		command.JoinTo(*joinTo)
		return true
	}
	if *members {
		command.ShowMembers()
		return true
	}
	return false
}

func main() {
	flag.Parse()
	if commandService() {
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
	tcpService := services.NewTcpService(&ctx)
	httpService := services.NewHttpService(&ctx)

	// 核心binlog服务
	blog := binlog.NewBinlog(&ctx)
	// 注册tcp、http、websocket服务
	blog.BinlogHandler.RegisterService("tcp", tcpService)
	blog.BinlogHandler.RegisterService("http", httpService)
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
	fmt.Println("wing binlog service exit...")
}
