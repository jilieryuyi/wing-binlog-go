package services

import (
	"library/file"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"library/app"
)

// parse http config
// config file is config/http.toml, here use absolute path
// use for service_http.go NewHttpService and Reload
func getHttpConfig() (*HttpConfig, error) {
	var config HttpConfig
	configFile := app.ConfigPath + "/http.toml"
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
	configFile := app.ConfigPath + "/tcp.toml"
	if !file.Exists(configFile) {
		log.Warnf("config %s does not exists", configFile)
		return nil, ErrorFileNotFound
	}
	var tcpConfig TcpConfig
	if _, err := toml.DecodeFile(configFile, &tcpConfig); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	if tcpConfig.ServiceIp == "" {
		log.Panicf("service ip can not be empty (config file: %s)", configFile)
	}
	if tcpConfig.Port <= 0 {
		log.Panicf("service port can not be 0 (config file: %s)", configFile)
	}
	return &tcpConfig, nil
}

func pack(cmd int, msg string) []byte {
	m  := []byte(msg)
	l  := len(m)
	r  := make([]byte, l+6)
	cl := l + 2
	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 24)
	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], m)
	return r
}