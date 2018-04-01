package http

import (
	"sync"
	"library/app"
	"time"
	"library/services"
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
	services.Service                                //
	groups           httpGroups//map[string]*httpGroup //
	lock             *sync.Mutex           // 互斥锁，修改资源时锁定
	timeTick         time.Duration         // 故障检测的时间间隔
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
	_ services.Service = &HttpService{}
)


