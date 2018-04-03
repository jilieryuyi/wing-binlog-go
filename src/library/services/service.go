package services

type Service interface {
	// 服务广播
	SendAll(table string, data []byte) bool
	// 启动服务
	Start()
	// 关闭服务
	Close()
	// 重新加载服务配置
	Reload()
	// 返回服务名称
	Name() string
	// 用于发送agent透传的原始数据
	// 目前只有service_plugin/tcp用到了
	SendRaw(data []byte) bool
}