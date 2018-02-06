package app

import (
	"github.com/BurntSushi/toml"
	log "github.com/jilieryuyi/logrus"
	"library/file"
	"library/path"
	"context"
)

type AppConfig struct {
	LogLevel int `toml:"log_level"`
	PprofListen string `toml:"pprof_listen"`
	TimeZone string `toml:"time_zone"`
}

var DEBUG = false

type Context struct {
	Ctx context.Context
	Cancel context.CancelFunc
	PidFile string
	ServiceIp string
	ServicePort int
	//Binlog *binlog.Binlog
}

type tcpGroupConfig struct { // group node in toml
	Name   string   // = "group1"
	Filter []string //
}

type TcpConfig struct {
	Listen string `toml:"listen"`
	Port   int    `toml:"port"`
	Enable bool   `toml:"enable"`
	ServiceIp string `toml:"service_ip"`
	Groups map[string]tcpGroupConfig
}

func GetTcpConfig() (*TcpConfig, error) {
	var tcp_config TcpConfig
	tcp_config_file := path.CurrentPath + "/config/tcp.toml"
	wfile := file.WFile{tcp_config_file}
	if !wfile.Exists() {
		log.Warnf("配置文件%s不存在 %s", tcp_config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(tcp_config_file, &tcp_config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &tcp_config, nil
}

func NewContext() *Context {
	config, _ := GetTcpConfig()
	ctx := &Context{
		ServicePort : config.Port,
		ServiceIp   : config.ServiceIp,
	}
	ctx.Ctx, ctx.Cancel = context.WithCancel(context.Background())
	return ctx
}

func GetAppConfig() (*AppConfig, error) {
	var appConfig AppConfig
	configFile := path.CurrentPath + "/config/wing-binlog-go.toml"
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
	return &appConfig, nil
}
