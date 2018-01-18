package app

import (
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"library/file"
	"library/path"
)

type AppConfig struct {
	LogLevel int `toml:"log_level"`
	PprofListen string `toml:"pprof_listen"`
	TimeZone string `toml:"time_zone"`
}

func GetAppConfig() (*AppConfig, error) {
	var appConfig AppConfig
	config_file := path.CurrentPath + "/config/wing-binlog-go.toml"
	wfile := file.WFile{config_file}
	if !wfile.Exists() {
		log.Errorf("配置文件%s不存在：%s", config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(config_file, &appConfig); err != nil {
		log.Errorf("读取配置文件错误：%+v", err)
		return nil, ErrorFileParse
	}
	//if appConfig.PprofListen == "" {
	//	appConfig.PprofListen = "0.0.0.0:6060"
	//}
	if appConfig.TimeZone == "" {
		appConfig.TimeZone = "Local"
	}
	return &appConfig, nil
}
