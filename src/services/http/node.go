package http

import (
	"sync"
	log "github.com/sirupsen/logrus"
	"library/http"
	"library/app"
	"runtime"
)

func newHttpNode(ctx *app.Context,url string) *httpNode {
	return &httpNode{
		url:              url,
		sendQueue:        make(chan string, httpMaxSendQueue),
		lock:             new(sync.Mutex),
		ctx:              ctx,
		wg:               new(sync.WaitGroup),
	}
}

func (node *httpNode) asyncSend(data []byte) {
	for {
		// if cache is full, try to wait it
		if len(node.sendQueue) < cap(node.sendQueue) {
			break
		}
		log.Warnf("cache full, try wait")
	}
	node.sendQueue <- string(data)
}

func (node *httpNode) wait() {
	node.wg.Wait()
}

func (node *httpNode) send(data []byte) ([]byte, error) {
	return http.Post(node.url, data)
}

func (nodes *httpNodes) asyncSend(data []byte) {
	for _, node := range *nodes {
		node.asyncSend(data)
	}
}

func (nodes *httpNodes) sendService() {
	cpu := runtime.NumCPU() + 2
	for _, node := range *nodes {
		// 启用cpu数量的服务协程
		for i := 0; i < cpu; i++ {
			go node.clientSendService()
		}
	}
}

func (node *httpNode) clientSendService() {
	node.wg.Add(1)
	defer node.wg.Done()
	for {
		select {
		case msg, ok := <-node.sendQueue:
			if !ok {
				log.Warnf("http service, sendQueue channel was closed")
				return
			}
			data, err := node.send([]byte(msg))
			if err != nil {
				log.Errorf("http service node %s error: %v", node.url, err)
			}
			log.Debugf("post to %s: %v return %s", msg, node.url, string(data))
		case <-node.ctx.Ctx.Done():
			l := len(node.sendQueue)
			log.Debugf("===>wait cache data post: %s left data len %d (if left data is 0, will exit) ", node.url, l)
			if l <= 0 {
				log.Debugf("%s clientSendService exit", node.url)
				return
			}
		}
	}
}
