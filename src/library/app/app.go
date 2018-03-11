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
	"strings"
	wstring "library/string"
	"library/ip"
)

var (
	Pid        = path.CurrentPath + "/wing-binlog-go.pid"
	DEBUG      = false
	ConfigPath = path.CurrentPath + "/config"
	CachePath  = path.CurrentPath + "/cache"
	LogPath    = path.CurrentPath + "/logs"
)

const (
	VERSION = "1.0.1"
)

type Config struct {
	LogLevel int       `toml:"log_level"`
	PprofListen string `toml:"pprof_listen"`
	TimeZone string    `toml:"time_zone"`
	CachePath string   `toml:"cache_path"`
	LogPath string     `toml:"log_path"`
	PidFile string     `toml:"pid_file"`
}

type HttpNodeConfig struct {
	Name   string
	Nodes  []string
	Filter []string
}

type HttpConfig struct {
	Enable   bool
	TimeTick time.Duration //故障检测的时间间隔，单位为秒
	Groups   map[string]HttpNodeConfig
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

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})

	log.SetLevel(log.DebugLevel) //log.DebugLevel)

	configPathParse(configPath)
	appConfig, _ := getAppConfig()
	log.SetLevel(log.Level(appConfig.LogLevel)) //log.DebugLevel)

	Pid = appConfig.PidFile
	CachePath = appConfig.CachePath
	LogPath   = appConfig.LogPath
	// set log context hook
	log.AddHook(mlog.ContextHook{LogPath:LogPath})
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

	// set cpu num
	cpu := runtime.NumCPU()
	runtime.GOMAXPROCS(cpu) //指定cpu为多核运行 旧版本兼容
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
	fmt.Println("wing-binlog-go                                     --start service")
	fmt.Println("wing-binlog-go -version                            --show version info")
	fmt.Println("wing-binlog-go -stop                               --stop service")
	fmt.Println("wing-binlog-go -reload  http                       --reload http service")
	fmt.Println("wing-binlog-go -reload  tcp                        --reload tcp service")
	fmt.Println("wing-binlog-go -reload  all                        --reload all service")
	fmt.Println("wing-binlog-go -members                            --show cluster members")
	fmt.Println("wing-binlog-go -d|-daemon                          --run as daemon process")
	fmt.Println("wing-binlog-go -config-path [path like: /tmp/wing] --set config path")
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

func getHttpConfig() (*HttpConfig, error) {
	var config HttpConfig
	configFile := ConfigPath + "/http.toml"
	if !file.Exists(configFile) {
		log.Warnf("config file %s does not exists", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	if config.TimeTick <= 0 {
		config.TimeTick = 1
	}
	return &config, nil
}

type TcpGroupConfigs map[string]TcpGroupConfig
func (cs *TcpGroupConfigs) HasName(name string) bool {
	for _, ngroup := range *cs {
		if name == ngroup.Name {
			return true
			break
		}
	}
	return false
}

type TcpConfig struct {
	Listen string `toml:"listen"`
	Port   int    `toml:"port"`
	Enable bool   `toml:"enable"`
	ServiceIp string `toml:"service_ip"`
	Groups TcpGroupConfigs
}

type TcpGroupConfig struct {
	Name   string
	Filter []string
}

func getTcpConfig() (*TcpConfig, error) {
	configFile := ConfigPath + "/tcp.toml"
	var err error
	if !file.Exists(configFile) {
		log.Warnf("config %s does not exists", configFile)
		return nil, ErrorFileNotFound
	}
	var tcpConfig TcpConfig
	if _, err = toml.DecodeFile(configFile, &tcpConfig); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	if 	tcpConfig.ServiceIp == "" {
		tcpConfig.ServiceIp, err = ip.Local()
		if err != nil {
			log.Panicf("can not get local ip, please set service ip(service_ip) in file %s", configFile)
		}
	}
	if tcpConfig.ServiceIp == "" {
		log.Panicf("service ip can not be empty (config file: %s)", configFile)
	}
	if tcpConfig.Port <= 0 {
		log.Panicf("service port can not be 0 (config file: %s)", configFile)
	}
	return &tcpConfig, nil
}

type MysqlConfig struct {
	// mysql service ip and port, like: "127.0.0.1:3306"
	Addr     string `toml:"addr"`
	// mysql service user
	User     string `toml:"user"`
	// mysql password
	Password string `toml:"password"`
	// mysql default charset
	Charset         string        `toml:"charset"`
	// mysql binlog client id, it must be unique
	ServerID        uint32        `toml:"server_id"`
	// mysql or mariadb
	Flavor          string        `toml:"flavor"`
	// heartbeat interval, unit is ns, 30000000000  = 30s   1000000000 = 1s
	HeartbeatPeriod time.Duration `toml:"heartbeat_period"`
	// read timeout, unit is ns, 0 is never timeout, 30000000000  = 30s   1000000000 = 1s
	ReadTimeout     time.Duration `toml:"read_timeout"`
	// read start form the binlog file
	BinFile string `toml:"bin_file"`
	// read start form the pos
	BinPos  uint32 `toml:"bin_pos"`
}

func getMysqlConfig() (*MysqlConfig, error) {
	var appConfig MysqlConfig
	configFile := ConfigPath + "/canal.toml"
	if !file.Exists(configFile) {
		log.Errorf("config file %s not found", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &appConfig); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &appConfig, nil
}

type ConsulConfig struct{
	Address string `toml:"address"`
}

// consul config
type ClusterConfig struct {
	Enable bool `toml:"enable"`
	Type string `toml:"type"`
	Lock string `toml:"lock"`
	Consul *ConsulConfig
}

func getClusterConfig() (*ClusterConfig, error) {
	var config ClusterConfig
	configFile := ConfigPath + "/cluster.toml"
	if !file.Exists(configFile) {
		log.Errorf("config file not found: %s", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &config, nil
}
