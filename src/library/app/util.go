package app

import (
	"library/path"
	"fmt"
	"os"
	"io/ioutil"
	"library/file"
	"net/http"
	_ "net/http/pprof"
	log "github.com/sirupsen/logrus"
	mlog "library/log"
	"time"
	"runtime"
)
var Pid = path.CurrentPath + "/wing-binlog-go.pid"
const (
	VERSION = "1.0.0"
)

func Init() {
	data := []byte(fmt.Sprintf("%d", os.Getpid()))
	ioutil.WriteFile(Pid, data, 0777)
	appConfig, _ := GetAppConfig()
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

	time.LoadLocation(appConfig.TimeZone)
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.AddHook(mlog.ContextHook{})
	log.SetLevel(log.Level(appConfig.LogLevel)) //log.DebugLevel)

	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu) //指定cpu为多核运行 旧版本兼容
}

func Release() {
	file.Delete(Pid)
}

func Usage() {
	fmt.Println("*********************************************************************")
	fmt.Println("wing-binlog-go                                   --start service")
	fmt.Println("wing-binlog-go -version                          --show version info")
	fmt.Println("wing-binlog-go -stop                             --stop service")
	fmt.Println("wing-binlog-go -service-reload  http             --reload http service")
	fmt.Println("wing-binlog-go -service-reload  tcp              --reload tcp service")
	fmt.Println("wing-binlog-go -service-reload  websocket        --reload websocket service")
	fmt.Println("wing-binlog-go -service-reload  all              --reload all service")
	fmt.Println("wing-binlog-go -members                          --show cluster members")
	fmt.Println("wing-binlog-go -d|-daemon                        --run as daemon process")
	fmt.Println("*********************************************************************")
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

