package services

import (
	log "github.com/sirupsen/logrus"
	"library/http"
	"runtime"
	"sync"
	"library/app"
)

func NewHttpService(ctx *app.Context) *HttpService {
	log.Debugf("start http service with config: %+v", ctx.HttpConfig)
	if !ctx.HttpConfig.Enable {
		return &HttpService{
			status: 0,
		}
	}
	gc := len(ctx.HttpConfig.Groups)
	client := &HttpService{
		lock:             new(sync.Mutex),
		groups:           make(map[string]*httpGroup, gc),
		status:           serviceEnable,
		timeTick:         ctx.HttpConfig.TimeTick,
		wg:               new(sync.WaitGroup),
		ctx:              ctx,
	}
	for _, groupConfig := range ctx.HttpConfig.Groups {
		client.groups[groupConfig.Name] = newHttpGroup(groupConfig)
	}
	return client
}

// 开始服务
func (client *HttpService) Start() {
	if client.status & serviceEnable <= 0 {
		return
	}
	cpu := runtime.NumCPU() + 2
	for _, group := range client.groups {
		for _, node := range group.nodes {
			// 启用cpu数量的服务协程
			for i := 0; i < cpu; i++ {
				client.wg.Add(1)
				go client.clientSendService(node)
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
				log.Errorf("http service node %s error: %v", node.url, err)
			}
			log.Debugf("post to %s: %v return %s", msg, node.url, string(data))
		case <-client.ctx.Ctx.Done():
			l := len(node.sendQueue)
			log.Debugf("===>wait cache data post: %s left data len %d (if left data is 0, will exit) ", node.url, l)
			if l <= 0 {
				log.Debugf("%s clientSendService exit", node.url)
				return
			}
		}
	}
}

func (client *HttpService) SendAll(table string, data []byte) bool {
	if client.status & serviceEnable <= 0 {
		return false
	}
	for _, cgroup := range client.groups {
		if cgroup.nodes == nil || len(cgroup.nodes) <= 0 ||
			!matchFilters(cgroup.filter, table) {
			continue
		}
		for _, cnode := range cgroup.nodes {
			log.Debugf("http send broadcast: %s=>%s", cnode.url, string(data))
			for {
				// if cache is full, try to wait it
				if len(cnode.sendQueue) < cap(cnode.sendQueue) {
					break
				}
				log.Warnf("cache full, try wait")
			}
			cnode.sendQueue <- string(data)
		}
	}
	return true
}

func (client *HttpService) syncSend(node *httpNode, data []byte) {
	data, err := http.Post(node.url, data)
	if err != nil {
		log.Warnf("http service node %s error: %v", node.url, err)
	}
	log.Debugf("http service post to %s return %s", node.url, string(data))
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
	client.ctx.ReloadHttpConfig()
	log.Debug("http service reloading...")
	client.status = 0
	if client.ctx.HttpConfig.Enable {
		client.status = serviceEnable
	}
	for name := range client.groups {
		delete(client.groups, name)
	}
	for _, cgroup := range client.ctx.HttpConfig.Groups {
		group := &httpGroup{
			name: cgroup.Name,
			filter: cgroup.Filter,
			nodes: make([]*httpNode, 0),
		}
		for i := range cgroup.Nodes {
			node := &httpNode{
				url:              cgroup.Nodes[i],
				sendQueue:        make(chan string, httpMaxSendQueue),
				lock:             new(sync.Mutex),
			}
			group.nodes = append(group.nodes, node)
		}
		client.groups[cgroup.Name] = group
	}
	log.Debug("http service reloaded.")
}

func (client *HttpService) AgentStart(serviceIp string, port int) {}
func (client *HttpService) AgentStop() {}
func (client *HttpService) SendPos(data []byte) {}