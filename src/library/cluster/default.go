package cluster

type Default struct{}

// 校验当前节点是否为leader，返回true，说明是leader，则当前节点开始工作
// 返回false，说明是follower
func (d *Default) Leader() bool {
	return true
}

// 同步游标相关信息
func (d *Default) Write(binFile string, pos int64, eventIndex int64) bool {
	return true
}

// 读取游标信息
// 返回binFile string, pos int64, eventIndex int64
func (d *Default) Read() (string, int64, int64) {
	return "", 0, 0
}

// 获取当前的集群成员
func (d *Default) Members() []ClusterMember {
	return nil
}
