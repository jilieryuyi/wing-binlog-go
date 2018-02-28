package services

import (
	"regexp"
	//log "github.com/sirupsen/logrus"
)

func newTcpGroup(group tcpGroupConfig) *tcpGroup {
	return &tcpGroup{
		name: group.Name,
		filter: group.Filter,
		nodes: nil,
	}
}

func (g *tcpGroup) match(table string) bool {
	if g.filter == nil || len(g.filter) <= 0 {
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
	g.nodes.append(node)// = append(g.nodes, node)
}

func (g *tcpGroup) remove(node *tcpClientNode) {
	g.nodes.remove(node)
	//for index, n := range g.nodes {
	//	if n == node {
	//		g.nodes = append(g.nodes[:index], g.nodes[index+1:]...)
	//		break
	//	}
	//}
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
