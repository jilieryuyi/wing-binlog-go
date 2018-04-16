package subscribe

import (
	"sync"
	"library/app"
	"net"
	"library/services"
	log "github.com/sirupsen/logrus"
)

type tcpGroups struct {
	g []*tcpClientNode
	lock *sync.Mutex
	ctx *app.Context
	unique int64
}

func newGroups(ctx *app.Context) *tcpGroups {
	g := &tcpGroups{
		unique:0,
		lock:new(sync.Mutex),
		g:make([]*tcpClientNode, 0),
		ctx: ctx,
	}
	return g
}

func (groups *tcpGroups) sendAll(table string, data []byte) bool {
	for _, group := range groups.g {
		log.Debugf("topics:%+v, %v", group.topics, table)
		// 如果有订阅主题
		if services.MatchFilters(group.topics, table) {
			group.asyncSend(data)
		}
	}
	return true
}

func (groups *tcpGroups) remove(node *tcpClientNode) {
	for index, n := range groups.g {
		if n == node {
			groups.g = append(groups.g[:index], groups.g[index+1:]...)
			break
		}
	}
}

func (groups *tcpGroups) reload() {
	groups.close()
}

func (groups *tcpGroups) asyncSend(data []byte) {
	for _, group := range groups.g {
		group.asyncSend(data)
	}
}

func (groups *tcpGroups) close() {
	for _, group := range groups.g {
		group.close()
	}
	groups.g = make([]*tcpClientNode, 0)
}


func (groups *tcpGroups) onConnect(conn *net.Conn) {
	node := newNode(groups.ctx, conn, NodeClose(groups.remove))
	groups.g = append(groups.g, node)
	go node.onConnect()
}
