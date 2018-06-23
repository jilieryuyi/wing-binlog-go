package agent

import (
	"library/app"
	"library/file"
	"github.com/BurntSushi/toml"
)

const (
	CMD_SET_PRO = iota // 注册客户端操作，加入到指定分组
	CMD_AUTH           // 认证（暂未使用）
	CMD_ERROR          // 错误响应
	CMD_TICK           // 心跳包
	CMD_EVENT          // 事件
	CMD_AGENT
	CMD_STOP
	CMD_RELOAD
	CMD_SHOW_MEMBERS
	CMD_POS
)

type OnPosFunc func(r []byte)
type AgentServerOption func(s *TcpService)

type Config struct {
	Enable bool          `toml:"enable"`
	Type string          `toml:"type"`
	Lock string          `toml:"lock"`
	AgentListen string   `toml:"agent_listen"`
	ConsulAddress string `toml:"consul_address"`
}

func getConfig() (*Config, error) {
	var config Config
	configFile := app.ConfigPath + "/agent.toml"
	if !file.Exists(configFile) {
		return nil, app.ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		return nil, app.ErrorFileParse
	}
	return &config, nil
}

