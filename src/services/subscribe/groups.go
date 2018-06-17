package subscribe

import (
	"sync"
	"library/app"
	"net"
	"library/service"
	log "github.com/sirupsen/logrus"
)

type tcpGroups struct {
	g []*tcpClientNode
	lock *sync.Mutex
	ctx *app.Context
	unique int64
	onRemove []OnRemoveFunc
}
type OnRemoveFunc func(conn *net.Conn)
type TcpGroupsOptions func(groups *tcpGroups)

func newGroups(ctx *app.Context, opts ...TcpGroupsOptions) *tcpGroups {
	g := &tcpGroups{
		unique:0,
		lock:new(sync.Mutex),
		g:make([]*tcpClientNode, 0),
		ctx: ctx,
		onRemove: make([]OnRemoveFunc, 0),
	}
	for _, f := range opts  {
		f(g)
	}
	return g
}

func SetOnRemove(f OnRemoveFunc) TcpGroupsOptions {
	return func(groups *tcpGroups) {
		groups.onRemove = append(groups.onRemove, f)
	}
}

func (groups *tcpGroups) sendAll(table string, data []byte) bool {
	for _, node := range groups.g {
		log.Debugf("topics:%+v, %v", node.topics, table)
		// 如果有订阅主题
		if service.MatchFilters(node.topics, table) {
			//log.Debugf("发送消息")
			node.asyncSend(data)
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
	for _, f:=range groups.onRemove {
		f(node.conn)
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
