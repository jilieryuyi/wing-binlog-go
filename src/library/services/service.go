package services

type Service interface {
	// 服务广播
	// 事件触发时会调用这个api
	SendAll(table string, data []byte) bool
	// 启动服务
	Start()
	// 关闭服务
	Close()
	// 重新加载服务配置
	Reload()
	// 返回服务名称
	Name() string
}