package services

import (
	"library/file"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"library/app"
	"library/ip"
	"regexp"
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
	var err error
	if !file.Exists(configFile) {
		log.Warnf("config %s does not exists", configFile)
		return nil, ErrorFileNotFound
	}
	var tcpConfig TcpConfig
	if _, err = toml.DecodeFile(configFile, &tcpConfig); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	if 	tcpConfig.ServiceIp == "" {
		tcpConfig.ServiceIp, err = ip.Local()
		if err != nil {
			log.Panicf("can not get local ip, please set service ip(service_ip) in file %s", configFile)
		}
	}
	if tcpConfig.ServiceIp == "" {
		log.Panicf("service ip can not be empty (config file: %s)", configFile)
	}
	if tcpConfig.Port <= 0 {
		log.Panicf("service port can not be 0 (config file: %s)", configFile)
	}
	return &tcpConfig, nil
}

func pack(cmd int, msg []byte) []byte {
	//m  := []byte(msg)
	l  := len(msg)
	r  := make([]byte, l+6)
	cl := l + 2
	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 24)
	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], msg)
	return r
}

//func unpack(data []byte) (int, string) {
//	clen := int(data[0]) | int(data[1]) << 8 |
//		int(data[2]) << 16 | int(data[3]) << 24
//	cmd  := int(data[4]) | int(data[5]) << 8
//	content := string(data[6 : clen + 4])
//	return cmd, content
//}

func hasCmd(cmd int) bool {
	return cmd == CMD_SET_PRO ||
		cmd == CMD_AUTH ||
		cmd == CMD_ERROR||
		cmd == CMD_TICK ||
		cmd == CMD_EVENT||
		cmd == CMD_AGENT||
		cmd == CMD_STOP||
		cmd == CMD_RELOAD||
		cmd == CMD_SHOW_MEMBERS||
		cmd == CMD_POS //||
		//cmd == CMD_PING
}

func matchFilters(filters []string, table string) bool {
	if filters == nil || len(filters) <= 0 {
		return true
	}
	for _, f := range filters {
		match, err := regexp.MatchString(f, table)
		if match && err == nil {
			return true
		}
	}
	return false
}


func PackPro(flag int, content []byte) []byte {
	// 数据打包
	l := len(content) + 3
	log.Debugf("PackPro len: %d", l)
	r := make([]byte, l + 4)
	// 4字节数据包长度
	r[0] = byte(l)
	r[1] = byte(l >> 8)
	r[2] = byte(l >> 16)
	r[3] = byte(l >> 24)
	// 2字节cmd
	r[4] = byte(CMD_SET_PRO)
	r[5] = byte(CMD_SET_PRO >> 8)
	r[6] = byte(flag)
	// 实际数据内容
	r = append(r[:7], content...)
	return r
}
