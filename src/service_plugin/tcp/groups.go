package tcp

import (
	"sync"
	"library/app"
	"net"
)

type tcpGroups struct {
	g map[string]*tcpGroup
	lock *sync.Mutex
	ctx *app.Context
}

func newGroups(ctx *app.Context) *tcpGroups {
	g := &tcpGroups{
		lock:new(sync.Mutex),
		g:make(map[string]*tcpGroup),
		ctx: ctx,
	}
	for _, group := range ctx.TcpConfig.Groups{
		tcpGroup := newTcpGroup(group)
		g.lock.Lock()
		g.add(tcpGroup)
		g.lock.Unlock()
	}
	return g
}

func (groups *tcpGroups) add(group *tcpGroup) {
	groups.lock.Lock()
	groups.g[group.name] = group
	groups.lock.Unlock()
}

func (groups *tcpGroups) delete(group *tcpGroup) {
	groups.lock.Lock()
	delete(groups.g, group.name)
	groups.lock.Unlock()
}

func (groups *tcpGroups) hasName(findName string) bool {
	groups.lock.Lock()
	_, ok := groups.g[findName]
	groups.lock.Unlock()
	return ok
	//for name := range groups.g {
	//	if name == findName {
	//		return true
	//		break
	//	}
	//}
	//return false
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
}

func (groups *tcpGroups) removeNode(node *tcpClientNode) {
	groups.lock.Lock()
	if group, found := groups.g[node.group]; found {
		group.remove(node)
	}
	groups.lock.Unlock()
}

func (groups *tcpGroups) addNode(node *tcpClientNode, groupName string) bool {
	groups.lock.Lock()
	group, found := groups.g[groupName]
	groups.lock.Unlock()
	if !found || groupName == "" {
		return false
	}
	group.append(node)
	return true
}

func (groups *tcpGroups) sendAll(table string, data []byte) bool {
	for _, group := range groups.g {
		// check if match
		if group.match(table) {
			group.asyncSend(data)
		}
	}
	return true
}

func (groups *tcpGroups) onConnect(conn *net.Conn) {
	node := newNode(groups.ctx, conn, NodeClose(groups.removeNode), NodePro(groups.addNode))
	go node.onConnect()
}

//func (groups *tcpGroups) sendRaw(msg []byte) {
//	groups.asyncSend(msg)
//}

