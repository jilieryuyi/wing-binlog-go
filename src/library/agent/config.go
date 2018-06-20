package agent

import (
	"library/app"
	"library/file"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

const (
	tcpNodeOnline = 1 << iota
)

type NodeFunc func(n *tcpClientNode)
type NodeOption func(n *tcpClientNode)

type tcpClients []*tcpClientNode

type OnPosFunc func(r []byte)
type AgentServerOption func(s *TcpService)
//var (
//	packDataTickOk     = service.Pack(CMD_TICK, []byte("ok"))
//	packDataSetPro     = service.Pack(CMD_SET_PRO, []byte("ok"))
//)

type Config struct {
	Enable bool `toml:"enable"`
	Type string `toml:"type"`
	Lock string `toml:"lock"`
	AgentListen string `toml:"agent_listen"`
	ConsulAddress string `toml:"consul_address"`
}

func getConfig() (*Config, error) {
	var config Config
	configFile := app.ConfigPath + "/agent.toml"
	if !file.Exists(configFile) {
		log.Errorf("config file not found: %s", configFile)
		return nil, app.ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, app.ErrorFileParse
	}
	return &config, nil
}

