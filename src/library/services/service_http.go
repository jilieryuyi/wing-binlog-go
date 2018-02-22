package services

import (
	log "github.com/sirupsen/logrus"
	"library/http"
	"regexp"
	"runtime"
	"sync"
	"library/app"
	"encoding/json"
)

// 创建一个新的http服务
func NewHttpService(ctx *app.Context) *HttpService {
	config, _ := getHttpConfig()
	log.Debugf("start http service with config: %+v", config)
	status := serviceDisable
	if !config.Enable {
		return &HttpService{
			status: status,
		}
	}
	status ^= serviceDisable
	status |= serviceEnable
	gc := len(config.Groups)
	client := &HttpService{
		lock:             new(sync.Mutex),
		groups:           make(map[string]*httpGroup, gc),
		sendFailureTimes: int64(0),
		status:           status,
		timeTick:         config.TimeTick,
		wg:               new(sync.WaitGroup),
		ctx:              ctx,
	}
	for _, cgroup := range config.Groups {
		group := &httpGroup{
			name: cgroup.Name,
		}
		group.filter = make([]string, len(cgroup.Filter))
		group.filter = append(group.filter[:0], cgroup.Filter...)
		nc := len(cgroup.Nodes)
		group.nodes = make([]*httpNode, nc)
		for i := 0; i < nc; i++ {
			group.nodes[i] = &httpNode{
				url:              cgroup.Nodes[i],
				sendQueue:        make(chan string, TCP_MAX_SEND_QUEUE),
				sendTimes:        int64(0),
				sendFailureTimes: int64(0),
				lock:             new(sync.Mutex),
				failureTimesFlag: int32(0),
				errorCheckTimes:  int64(0),
				status:           online | cacheNotReady | cacheNotFull,
			}
		}
		client.groups[cgroup.Name] = group
	}

	return client
}

// 开始服务
func (client *HttpService) Start() {
	if client.status & serviceDisable > 0 {
		return
	}
	cpu := runtime.NumCPU()
	for _, cgroup := range client.groups {
		for _, cnode := range cgroup.nodes {
			// 启用cpu数量的服务协程
			for i := 0; i < cpu; i++ {
				client.wg.Add(1)
				go client.clientSendService(cnode)
			}
		}
	}
}

// 节点服务协程
func (client *HttpService) clientSendService(node *httpNode) {
	defer client.wg.Done()
	for {
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Warnf("http service, sendQueue channel was closed")
				return
			}
			data, err := http.Post(node.url, []byte(msg))
			if err != nil {
				log.Warnf("http service node %s failure times：%d", node.url, node.sendFailureTimes)
			}
			log.Debugf("http service post to %s return %s", node.url, string(data))
		case <-client.ctx.Ctx.Done():
			if len(node.sendQueue) <= 0 {
				log.Debugf("http服务clientSendService退出：%s", node.url)
				return
			}
		}
	}
}

func (client *HttpService) SendAll(data map[string] interface{}) bool {
	if client.status & serviceDisable > 0 {
		return false
	}
	client.lock.Lock()
	defer client.lock.Unlock()

	for _, cgroup := range client.groups {
		if len(cgroup.nodes) <= 0 {
			continue
		}
		// length, 2 bytes
		//tableLen := int(msg[0]) + int(msg[1]<<8)
		// content
		table := data["table"].(string)//string(msg[2 : tableLen+2])
		jsonData, _:= json.Marshal(data)
		// check if the table name matches the filter
		if len(cgroup.filter) > 0 {
			found := false
			for _, f := range cgroup.filter {
				match, err := regexp.MatchString(f, table)
				if err != nil {
					continue
				}
				if match {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		for _, cnode := range cgroup.nodes {
			log.Debugf("http send broadcast: %s=>%s", cnode.url, string(jsonData))
			if len(cnode.sendQueue) >= cap(cnode.sendQueue) {
				log.Warnf("http send buffer full(weight):%s, %s", cnode.url, string(jsonData))
				continue
			}
			cnode.sendQueue <- string(jsonData)
		}
	}

	return true
}

func (client *HttpService) Close() {
	log.Debug("http service closing, waiting for buffer send complete.")
	for _, cgroup := range client.groups {
		if len(cgroup.nodes) > 0 {
			client.wg.Wait()
			break
		}
	}
	log.Debug("http service closed.")
}

func (client *HttpService) Reload() {
	config, _ := getHttpConfig()
	log.Debug("http service reloading...")

	status := serviceDisable
	if config.Enable {
		status ^= serviceDisable
		status |= serviceEnable
	}


	client.status = status//config.Enable
	for name := range client.groups {
		delete(client.groups, name)
	}

	for _, cgroup := range config.Groups {
		group := &httpGroup{
			name: cgroup.Name,
		}
		group.filter = make([]string, len(cgroup.Filter))
		group.filter = append(group.filter[:0], cgroup.Filter...)

		nc := len(cgroup.Nodes)
		group.nodes = make([]*httpNode, nc)
		for i := 0; i < nc; i++ {
			group.nodes[i] = &httpNode{
				url:              cgroup.Nodes[i],
				sendQueue:        make(chan string, TCP_MAX_SEND_QUEUE),
				sendTimes:        int64(0),
				sendFailureTimes: int64(0),
				lock:             new(sync.Mutex),
				failureTimesFlag: int32(0),
				errorCheckTimes:  int64(0),
				status:           online | cacheNotReady | cacheNotFull,
			}
		}
		client.groups[cgroup.Name] = group
	}
	log.Debug("http service reloaded.")
}

func (client *HttpService) AgentStart(serviceIp string, port int) {

}

func (client *HttpService) AgentStop() {

}