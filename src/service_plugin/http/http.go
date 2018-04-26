package http

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"library/app"
)

func NewHttpService(ctx *app.Context) *HttpService {
	config, err := getHttpConfig()
	if err != nil {
		log.Panicf("%v", err)
	}
	log.Debugf("start http service with config: %+v", config)
	if !config.Enable {
		return &HttpService{
			status: 0,
		}
	}
	gc := len(config.Groups)
	client := &HttpService{
		lock:             new(sync.Mutex),
		groups:           make(httpGroups, gc),
		status:           serviceEnable,
		timeTick:         config.TimeTick,
		ctx:              ctx,
	}
	for _, groupConfig := range config.Groups {
		httpGroup := newHttpGroup(ctx, groupConfig)
		client.lock.Lock()
		client.groups.add(httpGroup)
		client.lock.Unlock()
	}
	return client
}

// 开始服务
func (client *HttpService) Start() {
	if client.status & serviceEnable <= 0 {
		return
	}
	client.groups.sendService()
}

func (client *HttpService) SendAll(table string, data []byte) bool {
	if client.status & serviceEnable <= 0 {
		return false
	}
	client.groups.asyncSend(table, data)
	return true
}

func (client *HttpService) Close() {
	log.Debug("http service closing, waiting for buffer send complete.")
	client.groups.wait()
	log.Debug("http service closed.")
}

func (client *HttpService) Reload() {
	config, err := getHttpConfig()
	if err != nil {
		return
	}
	log.Debug("http service reloading...")
	client.status = 0
	if config.Enable {
		client.status = serviceEnable
	}
	for _, group := range client.groups {
		client.groups.delete(group)
	}
	for _, groupConfig := range config.Groups {
		httpGroup := newHttpGroup(client.ctx, groupConfig)
		client.lock.Lock()
		client.groups.add(httpGroup)
		client.lock.Unlock()
	}
	log.Debug("http service reloaded.")
}

func (client *HttpService) Name() string{
	return "http"
}