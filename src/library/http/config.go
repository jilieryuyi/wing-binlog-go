package http

import (
    "github.com/BurntSushi/toml"
    "library/file"
    log "github.com/sirupsen/logrus"
    "errors"
)

var (
    ErrorFileNotFound = errors.New("config file not fount")
    ErrorFileParse = errors.New("parse config error")
)

type tcpc struct {
    Listen string
    Port int
}

type websocketc struct {
    Listen string
    Port int
}

type http_service_config struct{
    Http tcpc
    Websocket websocketc
}

const (
    CMD_SET_PRO = 1 // 注册客户端操作，加入到指定分组
    CMD_AUTH    = 2 // 认证（暂未使用）
    CMD_OK      = 3 // 正常响应
    CMD_ERROR   = 4 // 错误响应
    CMD_TICK    = 5 // 心跳包
    CMD_EVENT   = 6 // 事件
    CMD_CONNECT = 7
    CMD_RELOGIN = 8
)

const (
    TCP_MAX_SEND_QUEUE            = 128 //100万缓冲区
    TCP_DEFAULT_CLIENT_SIZE       = 64
    TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
    TCP_RECV_DEFAULT_SIZE         = 4096
    TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
)

func getServiceConfig() (*http_service_config, error) {
    var tcp_config http_service_config
    config_file := file.GetCurrentPath()+"/config/admin.toml"
    wfile := file.WFile{config_file}
    if !wfile.Exists() {
        log.Printf("config file %s does not exists", config_file)
        return nil, ErrorFileNotFound
    }

    if _, err := toml.DecodeFile(config_file, &tcp_config); err != nil {
        log.Println(err)
        return nil, ErrorFileParse
    }
    return &tcp_config, nil
}


