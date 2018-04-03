package agent

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