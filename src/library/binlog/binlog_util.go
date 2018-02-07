package binlog

import (
	"library/path"
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"library/file"
	"library/app"
)

// 封包
//func pack(cmd int, client_id string, msgs []string) []byte {
//	client_id_len := len(client_id)
//	// 获取实际包长度
//	l := 0
//	for _, msg := range msgs {
//		l += len([]byte(msg)) + 4
//	}
//	// cl为实际的包内容长度，2字节cmd
//	cl := l + 2 + client_id_len
//	r := make([]byte, cl)
//	r[0] = byte(cmd)
//	r[1] = byte(cmd >> 8)
//	copy(r[2:], []byte(client_id))
//	base_start := 2 + client_id_len
//	for _, msg := range msgs {
//		m := []byte(msg)
//		ml := len(m)
//		// 前4字节存放长度
//		r[base_start+0] = byte(ml)
//		r[base_start+1] = byte(ml >> 8)
//		r[base_start+2] = byte(ml >> 16)
//		r[base_start+3] = byte(ml >> 24)
//		base_start += 4
//		// 实际的内容
//		copy(r[base_start:], m)
//		base_start += ml
//	}
//	return r
//}

func packPos(binFile string, pos int64, eventIndex int64) []byte {
	res := []byte(binFile)
	l := 16 + len(res)
	r := make([]byte, l + 2)
	// 2 bytes is data length
	r[0] = byte(l)
	r[1] = byte(l >> 8)
	// 8 bytes is pos
	r[2] = byte(pos)
	r[3] = byte(pos >> 8)
	r[4] = byte(pos >> 16)
	r[5] = byte(pos >> 24)
	r[6] = byte(pos >> 32)
	r[7] = byte(pos >> 40)
	r[8] = byte(pos >> 48)
	r[9] = byte(pos >> 56)
	// 8 bytes is event index
	r[10] = byte(eventIndex)
	r[11] = byte(eventIndex >> 8)
	r[12] = byte(eventIndex >> 16)
	r[13] = byte(eventIndex >> 24)
	r[14] = byte(eventIndex >> 32)
	r[15] = byte(eventIndex >> 40)
	r[16] = byte(eventIndex >> 48)
	r[17] = byte(eventIndex >> 56)
	// the last is binlog file
	r = append(r[:18], res...)
	return r
}

func unpackPos(data []byte) (string, int64, int64) {
	pos := int64(data[2]) | int64(data[3])<<8 | int64(data[4])<<16 |
		int64(data[5])<<24 | int64(data[6])<<32 | int64(data[7])<<40 |
		int64(data[8])<<48 | int64(data[9])<<56
	eventIndex := int64(data[10]) | int64(data[11])<<8 |
		int64(data[12])<<16 | int64(data[13])<<24 |
		int64(data[14])<<32 | int64(data[15])<<40 |
		int64(data[16])<<48 | int64(data[17])<<56
	return string(data[18:]), pos, eventIndex
}

func getConfig() (*Config, error) {
	var config Config
	configFile := path.CurrentPath + "/config/cluster.toml"
	//wfile := file.WFile{configFile}
	if !file.Exists(configFile) {
		log.Errorf("config file not found: %s", configFile)
		return nil, app.ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &config); err != nil {
		log.Println(err)
		return nil, app.ErrorFileParse
	}
	return &config, nil
}

// 获取mysql配置
func GetMysqlConfig() (*AppConfig, error) {
	var appConfig AppConfig
	configFile := path.CurrentPath + "/config/canal.toml"
	if !file.Exists(configFile) {
		log.Errorf("config file %s not found", configFile)
		return nil, app.ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &appConfig); err != nil {
		log.Println(err)
		return nil, app.ErrorFileParse
	}
	return &appConfig, nil
}


