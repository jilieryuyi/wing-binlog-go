package services
//如果当前客户端为follower
//agent会使用客户端连接到leader
//leader发生事件的时候通过agent转发到连接follower的tcp客户端
//实现数据代理

type Agent struct {
	tcp *TcpService
}
// return leader tcp service ip and port, like "127.0.0.1:9989"
func (ag *Agent) getLeader() string {
	return ""
}

func (ag *Agent) Start() {

}

func (ag *Agent) Close() {

}

func (ag *Agent) onMessage() {

}
