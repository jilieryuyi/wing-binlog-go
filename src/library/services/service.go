package services

type Service interface {
	SendAll(table string, data []byte) bool
	//SendPos(data []byte)
	Start()
	Close()
	Reload()
	//AgentStart(serviceIp string, port int)
	//AgentStop()
	Name() string
	SendRaw(data []byte) bool
}