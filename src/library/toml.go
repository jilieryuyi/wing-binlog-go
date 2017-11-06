package library

import (
    //"fmt"
    "github.com/BurntSushi/toml"
    "log"
    "github.com/juju/errors"
)

//import "time"
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
    Ignore_table []string// = ["Test.abc", "Test.123"]
    Bin_file string//     = ""
    Bin_pos int64//      = 0
}

type MysqlConfig struct {
    Host string//     = "127.0.0.1"
    User string//     = "root"
    Password string// = "123456"
    Port int//     = 3306
    Charset string//  = "utf8"
}

func (config *WConfig) Parse() (*AppConfig, error) {
    var app_config AppConfig

    wfile := WFile{config.Config_file}
    if !wfile.Exists() {
        log.Printf("config file %s does not exists", config.Config_file)
        return nil, ErrorFileNotFound
    }

    if _, err := toml.DecodeFile(config.Config_file, &app_config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }

    //fmt.Printf("Title: %s\n", config.Title)
    //fmt.Printf("Owner: %s (%s, %s), Born: %s\n",
    //    config.Owner.Name, config.Owner.Org, config.Owner.Bio,
    //    config.Owner.DOB)
    //fmt.Printf("Database: %s %v (Max conn. %d), Enabled? %v\n",
    //    config.DB.Server, config.DB.Ports, config.DB.ConnMax,
    //    config.DB.Enabled)
    //for serverName, server := range config.Servers {
    //    fmt.Printf("Server: %s (%s, %s)\n", serverName, server.IP, server.DC)
    //}
    //fmt.Printf("Client data: %v\n", config.Clients.Data)
    //fmt.Printf
    return &app_config, nil
}

