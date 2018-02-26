package binlog

import (
	"os"
	"sync"
	"time"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"github.com/hashicorp/consul/api"
	"library/app"
	"errors"
)

var (
	sessionEmpty = errors.New("session empty")
)
type AppConfig struct {
	// mysql service ip and port, like: "127.0.0.1:3306"
	Addr     string `toml:"addr"`
	// mysql service user
	User     string `toml:"user"`
	// mysql password
	Password string `toml:"password"`
	// mysql default charset
	Charset         string        `toml:"charset"`
	// mysql binlog client id, it must be unique
	ServerID        uint32        `toml:"server_id"`
	// mysql or mariadb
	Flavor          string        `toml:"flavor"`
	// heartbeat interval, unit is ns, 30000000000  = 30s   1000000000 = 1s
	HeartbeatPeriod time.Duration `toml:"heartbeat_period"`
	// read timeout, unit is ns, 0 is never timeout, 30000000000  = 30s   1000000000 = 1s
	ReadTimeout     time.Duration `toml:"read_timeout"`
	// read start form the binlog file
	BinFile string `toml:"bin_file"`
	// read start form the pos
	BinPos  uint32 `toml:"bin_pos"`
}

type Binlog struct {
	// github.com/siddontang/go-mysql interface
	canal.DummyEventHandler
	// config
	Config        *AppConfig
	// github.com/siddontang/go-mysql mysql protocol handler
	handler       *canal.Canal
	// context, like that use for wait coroutine exit
	ctx           *app.Context
	// use for wait coroutine exit
	wg            *sync.WaitGroup
	// lock
	lock          *sync.Mutex
	// event unique index
	EventIndex    int64
	// registered service, key is the name of the service
	services      map[string]services.Service
	// cache handler, use for read and write cache file
	// binlog_handler.go SaveBinlogPostionCache and getBinlogPositionCache
	cacheHandler  *os.File
	// the last read pos
	lastPos       uint32
	// the last read binlog file
	lastBinFile   string

	// consul lock key
	LockKey string
	// consul address
	Address string
	// tcp service ip
	ServiceIp string
	// tcp service port
	ServicePort int
	// consul session client
	Session *Session
	// unique session id
	sessionId string
	// consul client api
	Client *api.Client
	// consul kv service
	Kv *api.KV
	// consul agent, use for register service
	agent *api.Agent
	startServiceChan chan struct{}
	stopServiceChan chan bool
	posChan chan []byte
	// binlog status
	status int
}

const (
	// 针对停止服务 和 开始服务
	// binlog is stop service status
	binlogStatusIsStop = 1 << iota
	// binlog is start service status
	binlogStatusIsRunning
	// 最后两个状态成对
	// normal status
	binlogStatusIsNormal
	// binlog is in exit status, will exit later
	binlogStatusIsExit
	//针对binlog cache
	// binlog cache handler is closed
	cacheHandlerClosed
	// binlog cache handler is opened
	cacheHandlerOpened
	// consul is in locked status
	consulIsLeader//Lock
	// consul is in unlock status
	consulIsFollower
	// enable consul
	enableConsul
	// disable consul
	disableConsul
)
const (
	serviceKeepaliveTimeout  = 6  // timeout, unit is second
	checkAliveInterval       = 1  // interval for checkalive
	keepaliveInterval        = 1  // interval for keepalive

	prefixKeepalive = "wing/binlog/keepalive/"
	statusOnline    = "online"
	statusOffline   = "offline"
	serviceNameTcp  = "tcp"
	serviceNameHttp = "http"
	serviceNameAll  = "all"
	posChanLen      = 10000000
)

type ConsulConfig struct{
	Address string `toml:"address"`
}
// consul config
type Config struct {
	Enable bool `toml:"enable"`
	Type string `toml:"type"`
	Lock string `toml:"lock"`
	Consul *ConsulConfig
}

// cluster interface
type Cluster interface{
	Close()
	Lock() bool
	Write(data []byte) bool
	GetMembers() []*ClusterMember
	ClearOfflineMembers()
	GetServices() map[string]*api.AgentService
	GetLeader() (string, int)
}

// cluster node(member)
type ClusterMember struct {
	Hostname string
	IsLeader bool
	SessionId string
	Status string
	ServiceIp string
	Port int
}

// mysql config
type MysqlConfig struct {
	Addr string
	Port int
	User string
	Password string
	Database string
	Charset string
}




