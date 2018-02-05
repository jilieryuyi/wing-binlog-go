package cluster

import (
	"errors"
	"github.com/hashicorp/consul/api"
	"sync"
)

const (
	TIMEOUT  = 9 // timeout, unit is second
	CHECK_ALIVE_INTERVAL = 1 // interval for checkalive
	KEEPALIVE_INTERVAL = 3 // interval for keepalive

	POS_KEY          = "wing/binlog/pos"
	LOCK             = "wing/binlog/lock"
	PREFIX_KEEPALIVE = "wing/binlog/keepalive/"
	STATUS_ONLINE    = "online"
	STATUS_OFFLINE   = "offline"
)

type Consul struct {
	Cluster
	serviceIp string
	Session *Session
	isLock int
	lock *sync.Mutex
	onLeaderCallback func()
	onPosChange func([]byte)
	sessionId string
	enable bool
	Client *api.Client
	Kv *api.KV
	TcpServiceIp string
	TcpServicePort int
	agent *api.Agent
}

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

type ConsulConfig struct{
	ServiceIp string `toml:"service_ip"`
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

type Config struct {
	Enable bool `toml:"enable"`
	Type string `toml:"type"`
	Consul *ConsulConfig
}


