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

type ClusterMember struct {

}

type ConsulConfig struct{
	ServiceIp string
}
type MysqlConfig struct {
	Addr string//      = "127.0.0.1"
	Port int//      = 3306
	User string //      = "root"
	Password string//  = "123456"
	Database string//   = "wing-binlog-cluster"
	Charset string//   = "utf8"
}

type RedisConfig struct {
	Addr string// = "127.0.0.1"
	Port int// = 6379
}

type SsdbConfig struct {
	Addr string
	Port int
}

type Config struct {
	Enable bool `toml:"enable"`
	Type string `toml:"type"`
	Consul *ConsulConfig //`toml:"consul"`
	Mysql *MysqlConfig
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
