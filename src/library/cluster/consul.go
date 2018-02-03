package cluster

import (
	log "github.com/sirupsen/logrus"
	"time"
	"sync"
	"os"
	"github.com/hashicorp/consul/api"
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
	POS_KEY = "wing/binlog/pos"
	LOCK = "wing/binlog/lock"
	SESSION = "wing/binlog/session"
	PREFIX_KEEPALIVE = "wing/binlog/keepalive/"
	PREFIX_NODE = "wing/binlog/node/"
	STATUS_ONLINE = "online"
	STATUS_OFFLINE = "offline"
)

//todo 这里还缺一个服务注册与服务发现，健康检查
//服务注册-把当前的tcp service的服务ip和端口注册到consul
//服务发现即，查询这些服务并且分辨出那个是leader，那些节点是健康的
func NewConsul(onLeaderCallback func(), onPosChange func([]byte)) *Consul{
	log.Debugf("start cluster...")
	config, err := GetConfig()
	log.Debugf("cluster config: %+v", *config.Consul)
	if err != nil {
		log.Panicf("new consul client with error: %+v", err)
	}
	con := &Consul{
		serviceIp:config.Consul.ServiceIp,
		isLock:0,
		lock:new(sync.Mutex),
		sessionId:GetSession(),
		onLeaderCallback:onLeaderCallback,
		onPosChange:onPosChange,
		enable:config.Enable,
		TcpServiceIp:"",
		TcpServicePort:0,
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
			ID : "",
			s : con.Client.Session(),
		}
		con.Session.create()
		con.Kv = con.Client.KV()
		con.agent = con.Client.Agent()
		// check self is locked in start
		// if is locked, try unlock
		//v, _, err := con.Kv.Get(PREFIX_KEEPALIVE + con.sessionId, nil)
		//if err == nil && v != nil {
		//	t := int64(v.Value[0]) | int64(v.Value[1])<<8 |
		//		int64(v.Value[2])<<16 | int64(v.Value[3])<<24 |
		//		int64(v.Value[4])<<32 | int64(v.Value[5])<<40 |
		//		int64(v.Value[6])<<48 | int64(v.Value[7])<<56
		//	isLock := 0
		//	if len(v.Value) > 8 {
		//		isLock = int(v.Value[8])
		//	}
		//	if time.Now().Unix()-t > 3 && isLock == 1 {
		//		con.Unlock()
		//		con.Delete(LOCK)
		//		con.Delete(v.Key)
		//	}
		//}
		//超时检测，即检测leader是否挂了，如果挂了，要重新选一个leader
		//如果当前不是leader，重新选leader。leader不需要check
		//如果被选为leader，则还需要执行一个onLeader回调
		go con.checkAlive()
		////还需要一个keepalive
		go con.keepalive()
		//还需要一个检测pos变化回调，即如果不是leader，要及时更新来自leader的pos变化
		go con.watch()
	}
	return con
}

func (con *Consul) SetService(ip string, port int) {
	con.TcpServiceIp = ip
	con.TcpServicePort = port
}

// 注册服务
func (con *Consul) registerService() {
	con.lock.Lock()
	defer con.lock.Unlock()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	name := hostname + con.sessionId
	t := time.Now().Unix()
	r := make([]byte, 8)
	r[0] = byte(t)
	r[1] = byte(t >> 8)
	r[2] = byte(t >> 16)
	r[3] = byte(t >> 24)
	r[4] = byte(t >> 32)
	r[5] = byte(t >> 40)
	r[6] = byte(t >> 48)
	r[7] = byte(t >> 56)
	//log.Debugf("register time: %v", r)
	service := &api.AgentServiceRegistration{
		ID:                con.sessionId,
		Name:              name,
		Tags:              []string{string([]byte{byte(con.isLock)}), con.sessionId, string(r), hostname},
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
	//log.Debugf("services: %+v", ser)
	//debug
	//for key, v := range ser {
	//	log.Debugf("service %s: %+v", key, *v)
	//}
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
	members,_, err := con.Kv.List(PREFIX_KEEPALIVE, nil)
	if err != nil {
		return nil
	}
	_hostname, err := os.Hostname()
	log.Debugf("hostname: ", _hostname)
	m := make([]*ClusterMember, len(members))
	for i, v := range members {
		log.Debugf("key ==> %s", v.Key)
		m[i] = &ClusterMember{}
		t := int64(v.Value[0]) | int64(v.Value[1])<<8 |
			int64(v.Value[2])<<16 | int64(v.Value[3])<<24 |
			int64(v.Value[4])<<32 | int64(v.Value[5])<<40 |
			int64(v.Value[6])<<48 | int64(v.Value[7])<<56
		if time.Now().Unix()-t > 3 {
			m[i].Status = STATUS_OFFLINE
		} else {
			m[i].Status = STATUS_ONLINE
		}
		m[i].IsLeader = int(v.Value[8]) == 1
		//9 - 10 is hostname len
		hl := int(v.Value[9]) | int(v.Value[10])<<8
		//hl := int(member.Value[1]) | int(member.Value[2]) << 8
		m[i].Hostname = string(v.Value[11:12+hl])
		//kl := int(v.Value[hl+11]) | int(v.Value[hl+12]) << 8
		m[i].Session = string(v.Value[hl+13:])
	}
	return m
}

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
		//Tags: []string{string([]byte{byte(con.isLock)}), con.sessionId, string(t), hostname},
		for _, v := range services {
			//v0 := []byte(v.Tags[0])
			//isLock := int(v0[0]) == 1
			//sessionId := v.Tags[1]
			v2 := []byte(v.Tags[2])
			//log.Debugf("time: %v", v2)
			t := int64(v2[0]) | int64(v2[1])<<8 |
				int64(v2[2])<<16 | int64(v2[3])<<24 |
				int64(v2[4])<<32 | int64(v2[5])<<40 |
				int64(v2[6])<<48 | int64(v2[7])<<56
			//log.Debugf("now - t : %v - %v", time.Now().Unix(), t)
			if time.Now().Unix()-t > 3 {
				//m[i].Status = STATUS_OFFLINE
				//con.agent.ForceLeave()
				log.Warnf("%s is timeout, will be deregister", v.ID)
				con.agent.ServiceDeregister(v.ID)
			} //else {
			//m[i].Status = STATUS_ONLINE
			//}
			//hostName := v.Tags[3]
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

func (con *Consul) Read() []byte {
	if !con.enable {
		return nil
	}
	v ,_, err := con.Kv.Get(POS_KEY, nil)
	if err != nil {
		log.Errorf("write consul kv with error: %+v", err)
		return nil
	}
	return v.Value
}
