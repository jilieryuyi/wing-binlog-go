package cluster

import (
	"library/path"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"library/file"
)

func getConfig() (*Config, error) {
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
