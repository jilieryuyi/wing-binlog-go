package binlog

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"
	"unicode/utf8"
	"library/file"
	"library/path"
	"library/services"
	"github.com/BurntSushi/toml"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	log "github.com/sirupsen/logrus"
	"library/cluster"
)

var (
	ErrorFileNotFound = errors.New("文件不存在")
	ErrorFileParse    = errors.New("配置解析错误")
)

type AppConfig struct {
	Addr     string `toml:"addr"`
	User     string `toml:"user"`
	Password string `toml:"password"`

	Charset         string        `toml:"charset"`
	ServerID        uint32        `toml:"server_id"`
	Flavor          string        `toml:"flavor"`
	HeartbeatPeriod time.Duration `toml:"heartbeat_period"`
	ReadTimeout     time.Duration `toml:"read_timeout"`

	BinFile string `toml:"bin_file"`
	BinPos  uint32 `toml:"bin_pos"`
}

type Binlog struct {
	Config        *AppConfig
	handler       *canal.Canal
	isClosed      bool
	BinlogHandler *binlogHandler
	ctx           *context.Context
	wg            *sync.WaitGroup
	lock          *sync.Mutex
	isLeader      bool
	members       map[string]*member
	Drive         cluster.Cluster
}

type member struct {
	isLeader bool
	index    int
	status   string
}

type positionCache struct {
	pos   mysql.Position
	index int64
}

const (
	MEMBER_STATUS_LIVE  = "online"
	MEMBER_STATUS_LEAVE = "offline"

	MAX_CHAN_FOR_SAVE_POSITION = 128
	defaultBufSize             = 4096
	DEFAULT_FLOAT_PREC         = 6

	TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
	TCP_DEFAULT_CLIENT_SIZE       = 64
	TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
	TCP_RECV_DEFAULT_SIZE         = 4096
	TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
	CLUSTER_NODE_DEFAULT_SIZE     = 4

	CMD_APPEND_NODE   = 1
	CMD_POS           = 2
	CMD_JOIN          = 3
	CMD_GET_LEADER    = 4
	CMD_NEW_NODE      = 5
	CMD_KEEPALIVE     = 6
	CMD_CLOSE_CONFIRM = 7
	CMD_LEADER_CHANGE = 8
	CMD_NODE_SYNC = 9
)

type binlogHandler struct {
	EventIndex int64
	canal.DummyEventHandler
	buf           []byte
	services      map[string]services.Service
	servicesCount int
	cacheHandler  *os.File
	lock          *sync.Mutex // 互斥锁，修改资源时锁定
	isClosed      bool
	ctx           *context.Context
	lastPos       uint32
	lastBinFile   string
	binlog *Binlog
}

type Pos struct {
	BinFile string
	Pos int64
	EventIndex int64
}

// 获取mysql配置
func GetMysqlConfig() (*AppConfig, error) {
	var app_config AppConfig
	config_file := path.CurrentPath + "/config/canal.toml"
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

var htmlSafeSet = [utf8.RuneSelf]bool{
	' ':      true,
	'!':      true,
	'"':      false,
	'#':      true,
	'$':      true,
	'%':      true,
	'&':      false,
	'\'':     true,
	'(':      true,
	')':      true,
	'*':      true,
	'+':      true,
	',':      true,
	'-':      true,
	'.':      true,
	'/':      true,
	'0':      true,
	'1':      true,
	'2':      true,
	'3':      true,
	'4':      true,
	'5':      true,
	'6':      true,
	'7':      true,
	'8':      true,
	'9':      true,
	':':      true,
	';':      true,
	'<':      false,
	'=':      true,
	'>':      false,
	'?':      true,
	'@':      true,
	'A':      true,
	'B':      true,
	'C':      true,
	'D':      true,
	'E':      true,
	'F':      true,
	'G':      true,
	'H':      true,
	'I':      true,
	'J':      true,
	'K':      true,
	'L':      true,
	'M':      true,
	'N':      true,
	'O':      true,
	'P':      true,
	'Q':      true,
	'R':      true,
	'S':      true,
	'T':      true,
	'U':      true,
	'V':      true,
	'W':      true,
	'X':      true,
	'Y':      true,
	'Z':      true,
	'[':      true,
	'\\':     false,
	']':      true,
	'^':      true,
	'_':      true,
	'`':      true,
	'a':      true,
	'b':      true,
	'c':      true,
	'd':      true,
	'e':      true,
	'f':      true,
	'g':      true,
	'h':      true,
	'i':      true,
	'j':      true,
	'k':      true,
	'l':      true,
	'm':      true,
	'n':      true,
	'o':      true,
	'p':      true,
	'q':      true,
	'r':      true,
	's':      true,
	't':      true,
	'u':      true,
	'v':      true,
	'w':      true,
	'x':      true,
	'y':      true,
	'z':      true,
	'{':      true,
	'|':      true,
	'}':      true,
	'~':      true,
	'\u007f': true,
}
var hex = "0123456789abcdef"
