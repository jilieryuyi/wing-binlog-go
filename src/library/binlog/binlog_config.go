package binlog

import (
	"os"
	"sync"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"github.com/hashicorp/consul/api"
	"library/app"
	"errors"
)

var (
	sessionEmpty = errors.New("session empty")
	pingData = services.PackPro(services.FlagPing, []byte("ping"))
)

type Binlog struct {
	// github.com/siddontang/go-mysql interface
	canal.DummyEventHandler
	// config
	Config        *app.MysqlConfig
	// github.com/siddontang/go-mysql mysql protocol handler
	handler       *canal.Canal
	// context, like that use for wait coroutine exit
	ctx           *app.Context
	// use for wait coroutine exit
	wg            *sync.WaitGroup
	// lock
	lock          *sync.Mutex
	statusLock    *sync.Mutex
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
	//start stop
	_binlogIsRunning = 1 << iota
	// binlog is in exit status, will exit later
	_binlogIsExit
	_cacheHandlerIsOpened
	_consulIsLeader
	_enableConsul
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
	posChanLen      = 10000
)

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



