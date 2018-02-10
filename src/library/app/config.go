package app

import (
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"library/file"
	"library/path"
	"context"
	"os"
	"os/signal"
	"syscall"
	"strings"
)

type Config struct {
	LogLevel int       `toml:"log_level"`
	PprofListen string `toml:"pprof_listen"`
	TimeZone string    `toml:"time_zone"`
	CachePath string   `toml:"cache_path"`
	LogPath string     `toml:"log_path"`
}

// debug mode, default is false
var (
	DEBUG = false
    ConfigPath = path.CurrentPath + "/config"
    CachePath  = path.CurrentPath + "/cache"
    LogPath    = path.CurrentPath + "/logs"
)

// context
type Context struct {
	// canal context
	Ctx context.Context
	// canal context func
	Cancel context.CancelFunc
	// pid file path
	PidFile string
	CancelChan chan struct{}
}

// new app context
func NewContext() *Context {
	ctx := &Context{
		CancelChan:make(chan struct{}),
	}
	ctx.Ctx, ctx.Cancel = context.WithCancel(context.Background())
	go ctx.signalHandler()
	return ctx
}

func (ctx *Context)signalHandler() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sc
	ctx.CancelChan <- struct{}{}
}

func GetAppConfig() (*Config, error) {
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
	return &appConfig, nil
}
