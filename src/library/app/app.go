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
	"github.com/BurntSushi/toml"
	"context"
	"os/signal"
	"syscall"
	"strings"
	wstring "library/string"
)

var (
	Pid        = path.CurrentPath + "/wing-binlog-go.pid"
	DEBUG      = false
	ConfigPath = path.CurrentPath + "/config"
	CachePath  = path.CurrentPath + "/cache"
	LogPath    = path.CurrentPath + "/logs"
)

const (
	VERSION = "1.0.0"
)

type Config struct {
	LogLevel int       `toml:"log_level"`
	PprofListen string `toml:"pprof_listen"`
	TimeZone string    `toml:"time_zone"`
	CachePath string   `toml:"cache_path"`
	LogPath string     `toml:"log_path"`
	PidFile string     `toml:"pid_file"`
}

// context
type Context struct {
	// canal context
	Ctx context.Context
	// canal context func
	Cancel context.CancelFunc
	// pid file path
	PidFile string
	CancelChan chan struct{}
	ReloadChan chan string
	ShowMembersChan chan struct{}
	ShowMembersRes chan string
}

// app init
// config path parse
// cache path parse
// log path parse
// get app config
// check app is running, if pid file exists, app is running
// write pid file
// start pprof
// set logger
func Init(hasCmd bool, configPath string) {
	configPathParse(configPath)
	appConfig, _ := getAppConfig()
	Pid = appConfig.PidFile
	CachePath = appConfig.CachePath
	LogPath   = appConfig.LogPath

	if file.Exists(Pid) && !hasCmd {
		fmt.Println("other process still running")
		os.Exit(1)
	}

	// write pid file
	data := []byte(fmt.Sprintf("%d", os.Getpid()))
	ioutil.WriteFile(Pid, data, 0644)

	// run pprof
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
	// set timezone
	time.LoadLocation(appConfig.TimeZone)
	// set log format
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	// set log context hook
	log.AddHook(mlog.ContextHook{LogPath:LogPath})
	log.SetLevel(log.Level(appConfig.LogLevel)) //log.DebugLevel)
	// set cpu num
	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu) //指定cpu为多核运行 旧版本兼容

	log.Debugf("cache path: %s", CachePath)
	log.Debugf("log path: %s", LogPath)
	log.Debugf("app config: %+v", *appConfig)
}

// file path parse
func pathParse(dir string, defaultValue string) string {
	if dir == "" || !path.Exists(dir) {
		return defaultValue
	}
	dir = strings.Replace(dir, "\\", "/", -1)
	if dir[len(dir)-1:] == "/" {
		dir = dir[:len(dir)-1]
	}
	return dir
}

// config path parse
func configPathParse(configPath string) {
	ConfigPath = pathParse(configPath, ConfigPath)
	log.Debugf("load config form path: %s", ConfigPath)
}

// show usage
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
	fmt.Println("wing-binlog-go -config-path                      --set config path")
	fmt.Println("*********************************************************************")
}

// get unique key, param if file path
// if file does not exists, try to create it, and write a unique key
// return the unique key
// if exists, read file and return it
func GetKey(sessionFile string) string {
	//sessionFile := app.CachePath + "/session"
	log.Debugf("key file: %s", sessionFile)
	if file.Exists(sessionFile) {
		data := file.Read(sessionFile)
		if data != "" {
			return data
		}
	}
	//write a new key
	key := fmt.Sprintf("%d-%s", time.Now().Unix(), wstring.RandString(64))
	dir := path.GetParent(sessionFile)
	path.Mkdir(dir)
	n := file.Write(sessionFile, key, false)
	if n != len(key) {
		return ""
	}
	return key
}

// get app config
func getAppConfig() (*Config, error) {
	var appConfig Config
	configFile := ConfigPath + "/wing-binlog-go.toml"
	if !file.Exists(configFile) {
		log.Errorf("config file %s does not exists", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &appConfig); err != nil {
		log.Errorf("config file parse with error: %+v", err)
		return nil, ErrorFileParse
	}
	if appConfig.TimeZone == "" {
		appConfig.TimeZone = "Local"
	}

	appConfig.CachePath = strings.Trim(appConfig.CachePath, " ")
	if appConfig.CachePath != "" && !path.Exists(appConfig.CachePath) {
		path.Mkdir(appConfig.CachePath)
	}
	appConfig.CachePath = pathParse(appConfig.CachePath, CachePath)
	if appConfig.CachePath != "" && !path.Exists(appConfig.CachePath) {
		path.Mkdir(appConfig.CachePath)
	}

	appConfig.CachePath = strings.Trim(appConfig.CachePath," ")
	if appConfig.CachePath != "" && !path.Exists(appConfig.CachePath) {
		path.Mkdir(appConfig.CachePath)
	}
	appConfig.CachePath = pathParse(appConfig.CachePath, CachePath)
	if appConfig.CachePath != "" && !path.Exists(appConfig.CachePath) {
		path.Mkdir(appConfig.CachePath)
	}

	appConfig.LogPath = strings.Trim(appConfig.LogPath, " ")
	if appConfig.LogPath != "" && !path.Exists(appConfig.LogPath) {
		path.Mkdir(appConfig.LogPath)
	}
	appConfig.LogPath = pathParse(appConfig.LogPath, LogPath)
	if appConfig.LogPath != "" && !path.Exists(appConfig.LogPath) {
		path.Mkdir(appConfig.LogPath)
	}

	if appConfig.PidFile == "" {
		appConfig.PidFile = Pid
	} else {
		appConfig.PidFile = strings.Replace(appConfig.PidFile,"\\", "/", -1)
		dir := path.GetParent(appConfig.PidFile)
		path.Mkdir(dir)
	}

	return &appConfig, nil
}

// new app context
func NewContext() *Context {
	ctx := &Context{
		CancelChan:make(chan struct{}),
		ReloadChan:make(chan string, 100),
		ShowMembersChan:make(chan struct{}, 100),
		ShowMembersRes:make(chan string, 12),
	}
	ctx.Ctx, ctx.Cancel = context.WithCancel(context.Background())
	go ctx.signalHandler()
	return ctx
}

// wait for control + c signal
func (ctx *Context) signalHandler() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sc
	log.Warnf("get exit signal, service will exit later")
	ctx.CancelChan <- struct{}{}
}
