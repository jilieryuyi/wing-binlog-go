package cluster

import (
	"library/path"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"errors"
	"library/file"
)

var (
	ErrorFileNotFound = errors.New("config file not found")
	ErrorFileParse    = errors.New("config parse error")
)

type Cluster interface{
	Close()
	Lock() bool
	Write(data []byte) bool
	//RegisterOnLeaderCallback(fun func())
	//RegisterOnPosChangeCallback(fun func([]byte))
}

type ClusterMember struct {
	Hostname string
	IsLeader bool
	Updated int64
}

type ConsulConfig struct{
	ServiceIp string `toml:"service_ip"`
}
type MysqlConfig struct {
	Addr string
	Port int
	User string
	Password string
	Database string
	Charset string
}

type RedisConfig struct {
	Addr string
	Port int
}

type SsdbConfig struct {
	Addr string
	Port int
}

type Config struct {
	Enable bool `toml:"enable"`
	Type string `toml:"type"`
	Consul *ConsulConfig
	Redis *RedisConfig
	Ssdb *SsdbConfig
}

func GetConfig() (*Config, error) {
	var config Config
	configFile := path.CurrentPath + "/config/cluster.toml"
	wfile := file.WFile{configFile}
	if !wfile.Exists() {
		log.Errorf("config file not found: %s", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &config, nil
}
