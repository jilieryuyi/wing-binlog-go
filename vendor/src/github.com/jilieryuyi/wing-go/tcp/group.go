package tcp

import 	log "github.com/sirupsen/logrus"


func (c *Clients) append(node *ClientNode) {
	*c = append(*c, node)
}

func (c *Clients) send(msgId int64, data []byte) {
	for _, node := range *c {
		node.Send(msgId, data)
	}
}

func (c *Clients) asyncSend(msgId int64, data []byte) {
	for _, node := range *c {
		node.AsyncSend(msgId, data)
	}
}

func (c *Clients) remove(node *ClientNode) {
	for index, n := range *c {
		if n == node {
			*c = append((*c)[:index], (*c)[index+1:]...)
			break
		}
	}
	log.Debugf("#####################remove node, current len %v", len(*c))
}

func (c *Clients) close() {
	for _, node := range *c {
		node.close()
	}
}