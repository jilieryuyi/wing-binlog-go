package binlog

import (
	"os"
	"sync"
	"time"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"github.com/hashicorp/consul/api"
	"library/app"
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
	// binlog status, true means that was closed
	//isClosed      bool
	isRunning     int32 // > 0 is running
	// context, like that use for wait coroutine exit
	ctx           *app.Context
	// use for wait coroutine exit
	wg            *sync.WaitGroup
	// lock
	lock          *sync.Mutex
	// event unique index
	EventIndex int64
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
	// consul status, if == 1 means that current node is lock success, current is the leader
	isLock int
	// unique session id
	sessionId string
	// if true enable consul
	enable bool
	// consul client api
	Client *api.Client
	// consul kv service
	Kv *api.KV
	// consul agent, use for register service
	agent *api.Agent
}

const (
	serviceKeepaliveTimeout  = 6 // timeout, unit is second
	checkAliveInterval       = 1 // interval for checkalive
	keepaliveInterval        = 1 // interval for keepalive

	posKey          = "wing/binlog/pos/"
	prefixKeepalive = "wing/binlog/keepalive/"
	statusOnline    = "online"
	statusOffline   = "offline"
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




