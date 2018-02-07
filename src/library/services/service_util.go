package services

import (
	"library/file"
	"library/path"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

// parse http config
// config file is config/http.toml, here use absolute path
// use for service_http.go NewHttpService and Reload
func getHttpConfig() (*HttpConfig, error) {
	var config HttpConfig
	configFile := path.CurrentPath + "/config/http.toml"
	if !file.Exists(configFile) {
		log.Warnf("config file %s does not exists", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	if config.TimeTick <= 0 {
		config.TimeTick = 1
	}
	return &config, nil
}

func GetTcpConfig() (*TcpConfig, error) {
	configFile := path.CurrentPath + "/config/tcp.toml"
	if !file.Exists(configFile) {
		log.Warnf("config %s does not exists", configFile)
		return nil, ErrorFileNotFound
	}
	var tcpConfig TcpConfig
	if _, err := toml.DecodeFile(configFile, &tcpConfig); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &tcpConfig, nil
}
