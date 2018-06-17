package http

import (
	"sync"
	"library/app"
	"time"
	"library/service"
	"github.com/BurntSushi/toml"
	"library/file"
	log "github.com/sirupsen/logrus"
)

const (
	httpMaxSendQueue              = 10000
)

const (
	serviceEnable = 1 << iota
)

type httpGroup struct {
	name   string
	filter []string
	nodes  httpNodes//[]*httpNode
}

type httpNodes []*httpNode
type httpGroups map[string]*httpGroup

type HttpService struct {
	service.Service                //
	groups           httpGroups    //map[string]*httpGroup //
	lock             *sync.Mutex   // 互斥锁，修改资源时锁定
	timeTick         time.Duration // 故障检测的时间间隔
	ctx              *app.Context
	status           int
}

type httpNode struct {
	url              string      // url
	sendQueue        chan string // 发送channel
	lock             *sync.Mutex // 互斥锁，修改资源时锁定
	ctx              *app.Context
	wg               *sync.WaitGroup
}

var (
	_ service.Service = &HttpService{}
)


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

func getHttpConfig() (*HttpConfig, error) {
	var config HttpConfig
	configFile := app.ConfigPath + "/http.toml"
	if !file.Exists(configFile) {
		log.Warnf("config file %s does not exists", configFile)
		return nil, app.ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, app.ErrorFileParse
	}
	if config.TimeTick <= 0 {
		config.TimeTick = 1
	}
	return &config, nil
}

