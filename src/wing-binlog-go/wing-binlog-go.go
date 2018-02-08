package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	//"strconv"
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
	mlog "library/log"
	"library/path"
	"github.com/sevlyar/go-daemon"
)

var (
	//if debug is true, print stack log
	debug   = flag.Bool("debug", false, "enable debug, default disable")
	version = flag.Bool("version", false, "wing binlog go version")
	stop    = flag.Bool("stop", false, "stop service")
	//-service-reload http
	//-service-reload tcp
	//-service-reload all ##重新加载全部服务
	serviceReload  = flag.String("service-reload", "", "reload service config, usage: -service-reload  all|http|tcp")
	help           = flag.Bool("help", false, "help")
	members        = flag.Bool("members", false, "show members from current node")
	deamon         = flag.Bool("deamon", false, "-deamon or -d, run as deamon process")
	d              = flag.Bool("d", false, "-deamon or -d, run as deamon process")
	//clear          = flag.Bool("clear", false, "clear offline nodes from members, if need, it will auto register again")
)

const (
	VERSION = "1.0.0"
)

var (
	pid = path.CurrentPath + "/wing-binlog-go.pid"
 	appConfig, _ = app.GetAppConfig()
)

// write pid file
func writePid() {
	data := []byte(fmt.Sprintf("%d", os.Getpid()))
	ioutil.WriteFile(pid, data, 0777)
}

// delete pid file
func clearPid() {
	file.Delete(pid)
}

// kill process by pid file
//func killPid() {
//	dat, _ := ioutil.ReadFile(pid)
//	fmt.Print(string(dat))
//	pid, _ := strconv.Atoi(string(dat))
//	log.Debugf("try to kill process: %d", pid)
//	//err := syscall.Kill(pid, syscall.SIGTERM)
//	//log.Println(err)
//}

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
	fmt.Println("wing-binlog-go -members                          --show cluster members")
	fmt.Println("wing-binlog-go -clear                            --clear offline nodes from members, if need, it will auto register again")
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
	log.AddHook(mlog.ContextHook{})
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
	if *serviceReload != "" {
		command.Reload(*serviceReload)
		return true
	}
	// 帮助
	if *help {
		usage()
		return true
	}
	if *members {
		command.ShowMembers()
		return true
	}
	return false
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%+v", err)
		}
	}()
	flag.Parse()
	if commandService() {
		return
	}

	if *deamon || *d {
		cntxt := &daemon.Context{
			PidFileName: pid,
			PidFilePerm: 0644,
			LogFileName: path.CurrentPath + "/logs/wing-binlog-go.log",
			LogFilePerm: 0640,
			WorkDir:     path.CurrentPath,
			Umask:       027,
			Args:        []string{""},
		}
		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatal("Unable to run: ", err)
		}
		if d != nil {
			return
		}
		defer cntxt.Release()
	}

	app.DEBUG = *debug
	// 退出程序时删除pid文件
	defer clearPid()
	// 性能测试
	pprofService()
	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu) //指定cpu为多核运行 旧版本兼容

	appContext := app.NewContext()
	appContext.PidFile = pid

	// 各种通信服务
	httpService := services.NewHttpService(appContext)
	tcpService  := services.NewTcpService(appContext)

	// 核心binlog服务
	blog := binlog.NewBinlog(appContext)
	// 注册tcp、http、websocket服务
	blog.RegisterService("tcp", tcpService)
	blog.RegisterService("http", httpService)
	// here can be any drive interface form library.Cluster
	//blog.RegisterDrive(clu)
	blog.Start()

	// unix socket服务，用户本地指令控制
	server := unix.NewUnixServer(appContext, blog)
	server.Start()
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
	appContext.Cancel()
	blog.Close()
	fmt.Println("wing binlog service exit...")
}
