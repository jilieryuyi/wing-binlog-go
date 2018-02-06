package services

import (
	"sync"
	"time"
	"library/file"
	"library/path"
	"github.com/BurntSushi/toml"
	log "github.com/jilieryuyi/logrus"
	"library/app"
)

type httpNodeConfig struct {
	Name   string
	Nodes  []string
	Filter []string
}

type httpGroup struct {
	name   string      //
	filter []string    //
	nodes  []*httpNode //
}

type HttpService struct {
	Service                                //
	groups           map[string]*httpGroup //
	lock             *sync.Mutex           // 互斥锁，修改资源时锁定
	sendFailureTimes int64                 // 发送失败次数
	enable           bool                  //
	timeTick         time.Duration         // 故障检测的时间间隔
	ctx              *app.Context//*context.Context      //
	wg               *sync.WaitGroup       //
}

type HttpConfig struct {
	Enable   bool
	TimeTick time.Duration //故障检测的时间间隔，单位为秒
	Groups   map[string]httpNodeConfig
}

type httpNode struct {
	url              string      // url
	sendQueue        chan string // 发送channel
	sendTimes        int64       // 发送次数
	sendFailureTimes int64       // 发送失败次数
	isDown           bool        // 是否因为故障下线的节点
	failureTimesFlag int32       // 发送失败次数，用于配合last_error_time检测故障，故障定义为：连续三次发生错误和返回错误
	lock             *sync.Mutex // 互斥锁，修改资源时锁定
	cache            [][]byte
	cacheIndex       int
	cacheIsInit      bool
	cacheFull        bool
	errorCheckTimes  int64
}

func getHttpConfig() (*HttpConfig, error) {
	var config HttpConfig
	configFile := path.CurrentPath + "/config/http.toml"
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
