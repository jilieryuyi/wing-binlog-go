package cluster

import (
	log "github.com/sirupsen/logrus"
	"time"
	"sync"
	"os"
	"github.com/hashicorp/consul/api"
	"fmt"
	"strconv"
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

const (
	POS_KEY          = "wing/binlog/pos"
	LOCK             = "wing/binlog/lock"
	PREFIX_KEEPALIVE = "wing/binlog/keepalive/"
	STATUS_ONLINE    = "online"
	STATUS_OFFLINE   = "offline"
)

func NewConsul(onLeaderCallback func(), onPosChange func([]byte)) *Consul{
	config, err := GetConfig()
	if err != nil {
		log.Panicf("new consul client with error: %+v", err)
	}
	log.Debugf("cluster start with config: %+v", *config.Consul)
	con := &Consul {
		serviceIp        : config.Consul.ServiceIp,
		isLock           : 0,
		lock             : new(sync.Mutex),
		sessionId        : GetSession(),
		onLeaderCallback : onLeaderCallback,
		onPosChange      : onPosChange,
		enable           : config.Enable,
		TcpServiceIp     : "",
		TcpServicePort   : 0,
	}
	if con.enable {
		ConsulConfig := api.DefaultConfig()
		ConsulConfig.Address = config.Consul.ServiceIp
		var err error
		con.Client, err = api.NewClient(ConsulConfig)
		if err != nil {
			log.Panicf("create consul session with error: %+v", err)
		}
		con.Session = &Session {
			Address : config.Consul.ServiceIp,
			ID      : "",
			s       : con.Client.Session(),
		}
		con.Session.create()
		con.Kv = con.Client.KV()
		con.agent = con.Client.Agent()
		// check self is locked in start
		// if is locked, try unlock
		m := con.getService()
		if m != nil {
			if m.IsLeader && m.Status == STATUS_OFFLINE {
				log.Warnf("current node is lock in start, try to unlock")
				con.Unlock()
				con.Delete(LOCK)
			}
		}

		//超时检测，即检测leader是否挂了，如果挂了，要重新选一个leader
		//如果当前不是leader，重新选leader。leader不需要check
		//如果被选为leader，则还需要执行一个onLeader回调
		go con.checkAlive()
		//////还需要一个keepalive
		go con.keepalive()
		////还需要一个检测pos变化回调，即如果不是leader，要及时更新来自leader的pos变化
		go con.watch()
	}
	return con
}

func (con *Consul) SetService(ip string, port int) {
	con.TcpServiceIp = ip
	con.TcpServicePort = port
}

func (con *Consul) getService() *ClusterMember{
	members := con.GetMembers()
	for _, v := range members {
		if v.Session == con.sessionId {
			return v
		}
	}
	return nil
}

// register service
func (con *Consul) registerService() {
	con.lock.Lock()
	defer con.lock.Unlock()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	name := hostname + con.sessionId
	t := time.Now().Unix()
	il := []byte{byte(con.isLock)}
	service := &api.AgentServiceRegistration{
		ID:                con.sessionId,
		Name:              name,
		Tags:              []string{string(il), con.sessionId, fmt.Sprintf("%d", t), hostname},
		Port:              con.TcpServicePort,
		Address:           con.TcpServiceIp,
		EnableTagOverride: false,
		Check:             nil,
		Checks:            nil,
	}
	err = con.agent.ServiceRegister(service)
	if err != nil {
		log.Errorf("register service with error: %+v", err)
	}
}

// 服务发现，获取服务列表
func (con *Consul) GetServices() map[string]*api.AgentService {
	//1516574111-0hWR-E6IN-lrsO: {
	// ID:1516574111-0hWR-E6IN-lrsO
	// Service:yuyideMacBook-Pro.local1516574111-0hWR-E6IN-lrsO
	// Tags:[ 1516574111-0hWR-E6IN-lrsO /7tZ yuyideMacBook-Pro.local]
	// Port:9998 Address:127.0.0.1
	// EnableTagOverride:false
	// CreateIndex:0
	// ModifyIndex:0
	// }
	ser, err := con.agent.Services()
	if err != nil {
		log.Errorf("get service list error: %+v", err)
		return nil
	}
	return ser
}

func (con *Consul) keepalive() {
	for {
		con.Session.renew()
		con.registerService()
		time.Sleep(time.Second * 3)
	}
}

func (con *Consul) GetMembers() []*ClusterMember {
	members := con.GetServices()
	if members == nil {
		return nil
	}
	m := make([]*ClusterMember, len(members))
	var i int = 0
	for _, v := range members {
		m[i] = &ClusterMember{}
		t, _:= strconv.ParseInt(v.Tags[2], 10, 64)
		m[i].Status = STATUS_ONLINE
		if time.Now().Unix()-t > 3 {
			m[i].Status = STATUS_OFFLINE
		}
		m[i].IsLeader = int([]byte(v.Tags[0])[0]) == 1
		m[i].Hostname = v.Tags[3]
		m[i].Session = v.Tags[1]
		m[i].ServiceIp = v.Address
		m[i].Port = v.Port
	}
	return m
}

// check service is alive, is leader is not alive, try to select a new one
func (con *Consul) checkAlive() {
	if !con.enable {
		return
	}
	for {
		//获取所有的服务
		//判断服务的心跳时间是否超时
		//如果超时，更新状态为
		services := con.GetServices()
		if services == nil {
			time.Sleep(time.Second * 1)
			continue
		}
		for _, v := range services {
			isLock := int([]byte(v.Tags[0])[0]) == 1
			t, _ := strconv.ParseInt(v.Tags[2], 10, 64)
			if time.Now().Unix()-t > 3 {
				log.Warnf("%s is timeout, will be deregister", v.ID)
				con.agent.ServiceDeregister(v.ID)
				// if is leader, try delete lock and reselect a new leader
				if isLock {
					con.Delete(LOCK)
					if con.Lock() {
						log.Debugf("current is the new leader")
						if con.onLeaderCallback != nil {
							con.onLeaderCallback()
						}
					}
				}
			}
		}
		time.Sleep(time.Second * 1)
	}

	//for {
	//	con.lock.Lock()
	//	if con.isLock == 1 {
	//		con.lock.Unlock()
	//		// leader does not need check
	//		time.Sleep(time.Second * 3)
	//		continue
	//	}
	//	con.lock.Unlock()
	//	pairs, _, err := con.Kv.List(PREFIX_KEEPALIVE, nil)
	//	if err != nil {
	//		log.Errorf("checkAlive with error：%#v", err)
	//		time.Sleep(time.Second)
	//		continue
	//	}
	//	if pairs == nil {
	//		time.Sleep(time.Second * 3)
	//		continue
	//	}
	//	reLeader := true
	//	leaderCount := 0
	//	for _, v := range pairs {
	//		if v.Value == nil {
	//			log.Debugf("%+v", v)
	//			log.Debug("checkAlive value nil")
	//			continue
	//		}
	//		t := int64(v.Value[0]) | int64(v.Value[1]) << 8 |
	//				int64(v.Value[2]) << 16 | int64(v.Value[3]) << 24 |
	//				int64(v.Value[4]) << 32 | int64(v.Value[5]) << 40 |
	//				int64(v.Value[6]) << 48 | int64(v.Value[7]) << 56
	//		isLock := 0
	//		if len(v.Value) > 8 {
	//			isLock = int(v.Value[8])
	//		}
	//		if isLock == 1 {
	//			reLeader = false
	//			leaderCount++
	//		}
	//		if time.Now().Unix() - t > 3 {
	//			con.Delete(v.Key)
	//			if isLock == 1 {
	//				reLeader = true
	//			}
	//		}
	//	}
	//	if reLeader || leaderCount > 1 {
	//		log.Warnf("leader maybe leave, try to create a new leader")
	//		//con.Unlock()
	//		con.Delete(LOCK)
	//		if con.Lock() {
	//			if con.onLeaderCallback != nil {
	//				con.onLeaderCallback()
	//			}
	//		}
	//	}
	//	time.Sleep(time.Second * 3)
	//}
}

func (con *Consul) watch() {
	if !con.enable {
		return
	}
	for {
		con.lock.Lock()
		if con.isLock == 1 {
			con.lock.Unlock()
			// leader does not need watch
			time.Sleep(time.Second * 3)
			continue
		}
		con.lock.Unlock()
		_, meta, err := con.Kv.Get(POS_KEY, nil)
		if err != nil {
			log.Errorf("watch pos change with error：%#v", err)
			time.Sleep(time.Second)
			continue
		}
		if meta == nil {
			time.Sleep(time.Second)
			continue
		}
		v, _, err := con.Kv.Get(POS_KEY, &api.QueryOptions{
			WaitIndex : meta.LastIndex,
			WaitTime : time.Second * 3,
		})
		if err != nil {
			log.Errorf("watch chang with error：%#v, %+v", err, v)
			time.Sleep(time.Second)
			continue
		}
		if v == nil {
			time.Sleep(time.Second)
			continue
		}
		con.onPosChange(v.Value)
		time.Sleep(time.Millisecond * 10)
	}
}

func (con *Consul) GetLeader() (string, int) {
	members := con.GetMembers()
	for _, v := range members {
		if v.IsLeader {
			return v.ServiceIp, v.Port
		}
	}
	return "", 0
}

func (con *Consul) Close() {
	if !con.enable {
		return
	}
	con.Delete(PREFIX_KEEPALIVE + con.sessionId)
	log.Debugf("current is leader %d", con.isLock)
	con.lock.Lock()
	l := con.isLock
	con.lock.Unlock()
	if l == 1 {
		log.Debugf("delete lock %s", LOCK)
		con.Unlock()
		con.Delete(LOCK)
	}
	con.Session.delete()
}

func (con *Consul) Write(data []byte) bool {
	if !con.enable {
		return true
	}
	log.Debugf("write consul kv: %s, %v", POS_KEY, data)
	_, err := con.Kv.Put(&api.KVPair{Key: POS_KEY, Value: data}, nil)
	if err != nil {
		log.Errorf("write consul kv with error: %+v", err)
	}
	return nil == err
}
