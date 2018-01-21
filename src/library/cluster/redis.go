package cluster

type Redis struct {
	Cluster
}

func NewRedis() *Redis{
	return &Redis{}
}

func (redis *Redis) Start() {

}
func (redis *Redis) Close() {

}
func (redis *Redis) Write(data []byte) bool {
	return true
}
