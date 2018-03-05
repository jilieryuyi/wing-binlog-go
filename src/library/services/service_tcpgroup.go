package services

import (
	"regexp"
	"library/app"
)

func newTcpGroup(group app.TcpGroupConfig) *tcpGroup {
	g := &tcpGroup{
		name: group.Name,
		filter: group.Filter,
		nodes: nil,
	}
	return g
}

func (g *tcpGroup) match(table string) bool {
	if len(g.filter) <= 0 {
		return true
	}
	for _, f := range g.filter {
		match, err := regexp.MatchString(f, table)
		if match && err == nil {
			return true
		}
	}
	return false
}

func (g *tcpGroup) append(node *tcpClientNode) {
	g.nodes.append(node)
}

func (g *tcpGroup) remove(node *tcpClientNode) {
	g.nodes.remove(node)
}

func (g *tcpGroup) close() {
	for _, node := range g.nodes {
		node.close()
	}
}

func (c *tcpClients) append(node *tcpClientNode) {
	*c = append(*c, node)
}

func (c *tcpClients) send(data []byte) {
	for _, node := range *c {
		node.send(data)
	}
}

func (c *tcpClients) asyncSend(data []byte) {
	for _, node := range *c {
		node.asyncSend(data)
	}
}

func (c *tcpClients) remove(node *tcpClientNode) {
	for index, n := range *c {
		if n == node {
			*c = append((*c)[:index], (*c)[index+1:]...)
			break
		}
	}
}

func (c *tcpClients) close() {
	for _, node := range *c {
		node.close()
	}
}

func (g *tcpGroup) send(data []byte) {
	for _, node := range g.nodes {
		node.asyncSend(data)
	}
}
