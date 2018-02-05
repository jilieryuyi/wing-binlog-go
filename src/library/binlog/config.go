package binlog

import (
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
	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/consul/api"
	"library/app"
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
	ctx           *app.Context
	wg            *sync.WaitGroup
	lock          *sync.Mutex
	isLeader      bool
	members       map[string]*member

	EventIndex int64
	canal.DummyEventHandler
	buf           []byte
	services      map[string]services.Service
	cacheHandler  *os.File
	lastPos       uint32
	lastBinFile   string

	Address string   //consul address
	ServiceIp string //tcp service ip
	ServicePort int  //tcp service port
	Session *Session
	isLock int
	sessionId string
	enable bool
	Client *api.Client
	Kv *api.KV
	agent *api.Agent
}

type member struct {
	isLeader bool
	index    int
	status   string
}
//
//type positionCache struct {
//	pos   mysql.Position
//	index int64
//}

const (
	//MEMBER_STATUS_LIVE  = "online"
	//MEMBER_STATUS_LEAVE = "offline"
	//
	//MAX_CHAN_FOR_SAVE_POSITION = 128
	defaultBufferSize             = 4096
	//DEFAULT_FLOAT_PREC         = 6
	//
	//TCP_MAX_SEND_QUEUE            = 1000000 //100万缓冲区
	//TCP_DEFAULT_CLIENT_SIZE       = 64
	//TCP_DEFAULT_READ_BUFFER_SIZE  = 1024
	//TCP_RECV_DEFAULT_SIZE         = 4096
	//TCP_DEFAULT_WRITE_BUFFER_SIZE = 4096
	//CLUSTER_NODE_DEFAULT_SIZE     = 4
	//
	//CMD_APPEND_NODE   = 1
	//CMD_POS           = 2
	//CMD_JOIN          = 3
	//CMD_GET_LEADER    = 4
	//CMD_NEW_NODE      = 5
	//CMD_KEEPALIVE     = 6
	//CMD_CLOSE_CONFIRM = 7
	//CMD_LEADER_CHANGE = 8
	//CMD_NODE_SYNC = 9
)

//type binlogHandler struct {
//	EventIndex int64
//	canal.DummyEventHandler
//	buf           []byte
//	services      map[string]services.Service
//	//servicesCount int
//	cacheHandler  *os.File
//	lock          *sync.Mutex // 互斥锁，修改资源时锁定
//	isClosed      bool
//	ctx           *app.Context
//	lastPos       uint32
//	lastBinFile   string
//	binlog *Binlog
//}

type Pos struct {
	BinFile string
	Pos int64
	EventIndex int64
}

// 获取mysql配置
func GetMysqlConfig() (*AppConfig, error) {
	var appConfig AppConfig
	configFile := path.CurrentPath + "/config/canal.toml"
	if !file.Exists(configFile) {
		log.Errorf("config file %s not found", configFile)
		return nil, ErrorFileNotFound
	}
	if _, err := toml.DecodeFile(configFile, &appConfig); err != nil {
		log.Println(err)
		return nil, ErrorFileParse
	}
	return &appConfig, nil
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

type ConsulConfig struct{
	Address string `toml:"address"`
}

type Config struct {
	Enable bool `toml:"enable"`
	Type string `toml:"type"`
	Consul *ConsulConfig
}


const (
	serviceKeepaliveTimeout  = 9 // timeout, unit is second
	checkAliveInterval = 1 // interval for checkalive
	keepaliveInterval = 3 // interval for keepalive

	posKey          = "wing/binlog/pos"
	LOCK             = "wing/binlog/lock"
	prefixKeepalive = "wing/binlog/keepalive/"
	statusOnline    = "online"
	statusOffline   = "offline"
)
//
//type Consul struct {
//	Cluster
//	serviceIp string
//	tcp string
//	tcpPort int
//	Session *Session
//	isLock int
//	lock *sync.Mutex
//	//onLeaderCallback func()
//	//onPosChange func([]byte)
//	sessionId string
//	enable bool
//	Client *api.Client
//	Kv *api.KV
//	agent *api.Agent
//	ctx *app.Context
//	binlog *Binlog
//}

var (
	ErrorFileNotFound = errors.New("config file not found")
	ErrorFileParse    = errors.New("config parse error")
	ErrorSessionEmpty = errors.New("session empty")
)

type Cluster interface{
	Close()
	Lock() bool
	Write(data []byte) bool
	GetMembers() []*ClusterMember
	ClearOfflineMembers()
	GetServices() map[string]*api.AgentService
	GetLeader() (string, int)
}

type ClusterMember struct {
	Hostname string
	IsLeader bool
	Session string
	Status string
	ServiceIp string
	Port int
}

type MysqlConfig struct {
	Addr string
	Port int
	User string
	Password string
	Database string
	Charset string
}

type RedisConfig struct {
	Addr string
	Port int
}

type SsdbConfig struct {
	Addr string
	Port int
}




