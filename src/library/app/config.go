package app

import (
	"library/file"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)
type AppConfig struct {
	LogLevel int `toml:"log_level"`
}
func GetAppConfig() (*AppConfig, error) {
	var app_config AppConfig
	config_file := file.GetCurrentPath() + "/config/wing-binlog-go.toml"
	wfile := file.WFile{config_file}
	if !wfile.Exists() {
		log.Errorf("配置文件%s不存在：%s", config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(config_file, &app_config); err != nil {
		log.Errorf("读取配置文件错误：%+v", err)
		return nil, ErrorFileParse
	}
	return &app_config, nil
}
