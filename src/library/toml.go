package library

import (
    "github.com/BurntSushi/toml"
    "log"
    "github.com/juju/errors"
    "library/services"
    "library/file"
)

var (
    ErrorFileNotFound = errors.New("config file not fount")
    ErrorFileParse = errors.New("parse config error")
)

type WConfig struct {
    Config_file string
}

type AppConfig struct {
    Client   ClientConfig
    Mysql    MysqlConfig
    //DB      database `toml:"database"`
    //Servers map[string]server
    //Clients clients
}

type ClientConfig struct {
    Slave_id int
    Ignore_tables []string// = ["Test.abc", "Test.123"]
    Bin_file string//     = ""
    Bin_pos int64//      = 0
}

type MysqlConfig struct {
    Host string//     = "127.0.0.1"
    User string//     = "root"
    Password string// = "123456"
    Port int//     = 3306
    Charset string//  = "utf8"
    DbName string
}

// 获取mysql配置
func (config *WConfig) GetMysql() (*AppConfig, error) {
    var app_config AppConfig

    wfile := file.WFile{config.Config_file}
    if !wfile.Exists() {
        log.Printf("config file %s does not exists", config.Config_file)
        return nil, ErrorFileNotFound
    }

    if _, err := toml.DecodeFile(config.Config_file, &app_config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }
    return &app_config, nil
}

func (config *WConfig) GetTcp() (*services.TcpConfig, error) {
    var tcp_config services.TcpConfig

    wfile := file.WFile{config.Config_file}
    if !wfile.Exists() {
        log.Printf("config file %s does not exists", config.Config_file)
        return nil, ErrorFileNotFound
    }

    if _, err := toml.DecodeFile(config.Config_file, &tcp_config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }
    return &tcp_config, nil
}

func (config *WConfig) GetHttp() (*services.HttpConfig, error) {
    var tcp_config services.HttpConfig

    wfile := file.WFile{config.Config_file}
    if !wfile.Exists() {
        log.Printf("config file %s does not exists", config.Config_file)
        return nil, ErrorFileNotFound
    }

    if _, err := toml.DecodeFile(config.Config_file, &tcp_config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }
    return &tcp_config, nil
}
