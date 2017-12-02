package binlog

import (
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
	"errors"
	"library/file"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
)

var (
	ErrorFileNotFound = errors.New("文件不存在")
	ErrorFileParse = errors.New("配置解析错误")
)

type AppConfig struct {
	Client   ClientConfig
	Mysql    MysqlConfig
}

type ClientConfig struct {
	Slave_id int
	Ignore_tables []string
	Bin_file string
	Bin_pos int64
}

type MysqlConfig struct {
	Host string
	User string
	Password string
	Port int
	Charset string
	DbName string
}

type Binlog struct {
	DB_Config *AppConfig
	handler *canal.Canal
	is_connected bool
	binlog_handler binlogHandler
}

type positionCache struct {
	pos mysql.Position
	index int64
}

const (
	MAX_CHAN_FOR_SAVE_POSITION = 128
	defaultBufSize = 4096
	DEFAULT_FLOAT_PREC = 6
)

type binlogHandler struct {
	Event_index int64
	canal.DummyEventHandler
	chan_save_position chan positionCache
	buf               []byte
	tcp_service       *services.TcpService
	websocket_service *services.WebSocketService
	http_service      *services.HttpService
}

// 获取mysql配置
func GetMysqlConfig() (*AppConfig, error) {
	var app_config AppConfig
	config_file := file.GetCurrentPath() + "/config/mysql.toml"
	wfile := file.WFile{config_file}
	if !wfile.Exists() {
		log.Errorf("配置文件%s不存在 %s", config_file)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(config_file, &app_config); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &app_config, nil
}

