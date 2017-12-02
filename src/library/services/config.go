package services

import (
    "github.com/BurntSushi/toml"
    "library/file"
    log "github.com/sirupsen/logrus"
    "errors"
)

type tcpGroupConfig struct {
    Mode int     // "1 broadcast" ##(广播)broadcast or  2 (权重)weight
    Name string  // = "group1"
    Filter []string
}
type tcpConfig struct {
    Listen string
    Port int
}
type TcpConfig struct {
    Enable bool
    Groups map[string]tcpGroupConfig
    Tcp tcpConfig
}

type HttpConfig struct {
    Enable bool
    Groups map[string]httpNodeConfig
}

type httpNodeConfig struct {
    Mode int
    Nodes [][]string
    Filter []string
}

var (
    ErrorFileNotFound = errors.New("config file not fount")
    ErrorFileParse = errors.New("parse config error")
)

const (
    MODEL_BROADCAST = 1  // 广播
    MODEL_WEIGHT    = 2  // 权重

    CMD_SET_PRO = 1 // 注册客户端操作，加入到指定分组
    CMD_AUTH    = 2 // 认证（暂未使用）
    CMD_OK      = 3 // 正常响应
    CMD_ERROR   = 4 // 错误响应
    CMD_TICK    = 5 // 心跳包
    CMD_EVENT   = 6 // 事件

    TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
    TCP_DEFAULT_CLIENT_SIZE       = 64
    TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
    TCP_RECV_DEFAULT_SIZE         = 4096
    TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
)


func getTcpConfig() (*TcpConfig, error) {
    var tcp_config TcpConfig
    tcp_config_file := file.GetCurrentPath() + "/config/tcp.toml"
    wfile := file.WFile{tcp_config_file}
    if !wfile.Exists() {
        log.Printf("配置文件%s不存在", tcp_config_file)
        return nil, ErrorFileNotFound
    }
    if _, err := toml.DecodeFile(tcp_config_file, &tcp_config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }
    return &tcp_config, nil
}

func getHttpConfig() (*HttpConfig, error) {
    var config HttpConfig
    http_config_file := file.GetCurrentPath() + "/config/http.toml"
    wfile := file.WFile{http_config_file}
    if !wfile.Exists() {
        log.Printf("配置文件%s不存在", http_config_file)
        return nil, ErrorFileNotFound
    }
    if _, err := toml.DecodeFile(http_config_file, &config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }
    return &config, nil
}


func getWebsocketConfig() (*TcpConfig, error) {
    var config TcpConfig
    config_file := file.GetCurrentPath() + "/config/websocket.toml"
    wfile := file.WFile{config_file}
    if !wfile.Exists() {
        log.Printf("配置文件%s不存在", config_file)
        return nil, ErrorFileNotFound
    }
    if _, err := toml.DecodeFile(config_file, &config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }
    return &config, nil
}
