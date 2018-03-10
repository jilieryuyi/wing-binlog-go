package services

import (
	"sync"
)

func newHttpNode(url string) *httpNode {
	return &httpNode{
		url:              url,
		sendQueue:        make(chan string, httpMaxSendQueue),
		lock:             new(sync.Mutex),
	}
}
