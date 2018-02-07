package services

import (
	"errors"
)

type Service interface {
	SendAll(data map[string] interface{}) bool
	Start()
	Close()
	Reload()
	AgentStart(serviceIp string, port int)
	AgentStop()
}

var (
	ErrorFileNotFound = errors.New("config file not found")
	ErrorFileParse    = errors.New("config parse error")
)

const (
	CMD_SET_PRO = 1 // 注册客户端操作，加入到指定分组
	CMD_AUTH    = 2 // 认证（暂未使用）
	CMD_OK      = 3 // 正常响应
	CMD_ERROR   = 4 // 错误响应
	CMD_TICK    = 5 // 心跳包
	CMD_EVENT   = 6 // 事件
	CMD_AGENT   = 7

	TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
	TCP_DEFAULT_CLIENT_SIZE       = 64
	tcpDefaultReadBufferSize      = 1024
	tcpRecviveDefaultSize         = 4096

	HTTP_CACHE_LEN         = 10000
)
