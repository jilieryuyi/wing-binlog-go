package binlog

import (
	log "github.com/sirupsen/logrus"
	"time"
	"os"
	"github.com/hashicorp/consul/api"
	"fmt"
	"strconv"
	"net"
)

func (h *Binlog) consulInit() {
	var err error
	consulConfig, err := getConfig()

	h.LockKey     = consulConfig.Lock
	//consul config
	h.Address     = consulConfig.Consul.Address
	h.isLock      = 0
	h.sessionId   = GetSession()
	h.enable      = consulConfig.Enable

	ConsulConfig := api.DefaultConfig()
	ConsulConfig.Address = h.Address
	h.Client, err = api.NewClient(ConsulConfig)
	if err != nil {
		log.Panicf("create consul session with error: %+v", err)
	}

	h.Session = &Session {
		Address : h.Address,
		ID      : "",
		s       : h.Client.Session(),
	}
	h.Session.create()
	h.Kv    = h.Client.KV()
	h.agent = h.Client.Agent()
	// check self is locked in start
	// if is locked, try unlock
	m := h.getService()
	if m != nil {
		if m.IsLeader && m.Status == statusOffline {
			log.Warnf("current node is lock in start, try to unlock")
			h.Unlock()
			h.Delete(h.LockKey)
		}
	}
	// 超时检测，即检测leader是否挂了，如果挂了，要重新选一个leader
	// 如果当前不是leader，重新选leader。leader不需要check
	// 如果被选为leader，则还需要执行一个onLeader回调
	// check other is alive, if not, try to select a new leader
	go h.checkAlive()
	// 还需要一个keepalive
	// keepalive
	go h.keepalive()
	// 还需要一个检测pos变化回调，即如果不是leader，要及时更新来自leader的pos变化
	// watch pos change, if change, try to write cache
	go h.watch()
}

func (h *Binlog) getService() *ClusterMember{
	if !h.enable {
		return nil
	}
	members := h.GetMembers()
	if members == nil {
		return nil
	}
	for _, v := range members {
		if v != nil && v.SessionId == h.sessionId {
			return v
		}
	}
	return nil
}

// register service
func (h *Binlog) registerService() {
	if !h.enable {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	t := time.Now().Unix()
	service := &api.AgentServiceRegistration{
		ID:                h.sessionId,
		Name:              h.LockKey,
		Tags:              []string{fmt.Sprintf("%d", h.isLock), h.sessionId, fmt.Sprintf("%d", t), hostname, h.LockKey},
		Port:              h.ServicePort,
		Address:           h.ServiceIp,
		EnableTagOverride: false,
		Check:             nil,
		Checks:            nil,
	}
	err = h.agent.ServiceRegister(service)
	if err != nil {
		log.Errorf("register service with error: %+v", err)
	}
}

// keepalive
func (h *Binlog) keepalive() {
	if !h.enable {
		return
	}
	for {
		h.Session.renew()
		h.registerService()
		time.Sleep(time.Second * keepaliveInterval)
	}
}

// get all members nodes
func (h *Binlog) GetMembers() []*ClusterMember {
	if !h.enable {
		return nil
	}
	members, err := h.agent.Services()
	if err != nil {
		log.Errorf("get service list error: %+v", err)
		return nil
	}
	if members == nil {
		return nil
	}
	data := make([]*ClusterMember, 0)
	//fmt.Println("")
	//log.Debugf("============================================")
	for _, v := range members {
		// 这里的两个过滤，为了避免与其他服务冲突，只获取相同lockkey的服务，即 当前集群
		if len(v.Tags) < 5 {
			continue
		}
		if v.Tags[4] != h.LockKey {
			continue
		}
		m := &ClusterMember{}
		t, _:= strconv.ParseInt(v.Tags[2], 10, 64)
		m.Status = statusOnline
		if time.Now().Unix() - t > serviceKeepaliveTimeout {
			m.Status = statusOffline
		}
		m.IsLeader  = v.Tags[0] == "1"
		m.Hostname  = v.Tags[3]
		m.SessionId   = v.Tags[1]
		m.ServiceIp = v.Address
		m.Port      = v.Port
		data = append(data, m)
		//log.Debugf("member: %+v, %+v", *v, *m)
	}
	//log.Debugf("============================================")

	return data
}

// check service is alive
// if leader is not alive, try to select a new one
func (h *Binlog) checkAlive() {
	if !h.enable {
		return
	}
	for {
		//获取所有的服务
		//判断服务的心跳时间是否超时
		//如果超时，更新状态为
		members := h.GetMembers()
		if members == nil {
			time.Sleep(time.Second * checkAliveInterval)
			continue
		}
		for _, v := range members {
			if v.SessionId == h.sessionId {
				continue
			}
			//isLock := v.Tags[0] == "1"
			//t, _ := strconv.ParseInt(v.Tags[2], 10, 64)
			//only check other nodes is alive
			if v.Status == statusOffline {
				log.Warnf("%s is timeout, will be deregister", v.SessionId)
				h.agent.ServiceDeregister(v.SessionId)
				// if is leader, try delete lock and reselect a new leader
				//if is leader, ping, check leader is alive again
				if v.IsLeader && !h.alive(v.ServiceIp, v.Port) {
					h.Delete(h.LockKey)
					if h.Lock() {
						log.Debugf("current is the new leader")
						h.onNewLeader()
					}
				}
			}
		}
		time.Sleep(time.Second * checkAliveInterval)
	}
}

func (h *Binlog) alive(ip string, port int) bool {
	log.Debugf("ping %s:%d", ip, port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		log.Debugf("is not alive")
		return false
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil || conn == nil {
		log.Debugf("is not alive")
		return false
	}
	conn.Close()
	log.Debugf("is alive")
	return true
}

// watch pos change
// if pos write by other node
// all nodes will get change
func (h *Binlog) watch() {
	if !h.enable {
		return
	}
	for {
		h.lock.Lock()
		if h.isLock == 1 {
			h.lock.Unlock()
			// leader does not need watch
			time.Sleep(time.Second * 3)
			continue
		}
		h.lock.Unlock()
		_, meta, err := h.Kv.Get(posKey + h.LockKey, nil)
		if err != nil {
			log.Errorf("watch pos change with error：%#v", err)
			time.Sleep(time.Second)
			continue
		}
		if meta == nil {
			time.Sleep(time.Second)
			continue
		}
		v, _, err := h.Kv.Get(posKey + h.LockKey, &api.QueryOptions{
			WaitIndex : meta.LastIndex,
			WaitTime : time.Second * 86400,
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
		//h.onPosChange(v.Value)
		//h.ctx.PosChangeList <- v.Value
		h.onPosChange(v.Value)
		time.Sleep(time.Millisecond * 10)
	}
}

// get leader service ip and port
// if not found or some error happened
// return empty string and 0
func (h *Binlog) GetLeader() (string, int) {
	if !h.enable {
		log.Debugf("not enable")
		return "", 0
	}
	members := h.GetMembers()
	if members == nil || len(members) == 0 {
		return "", 0
	}
	for _, v := range members {
		log.Debugf("GetLeader--%+v", v)
		if v.IsLeader {
			return v.ServiceIp, v.Port
		}
	}
	return "", 0
}

// if app is close, it will be call for clear some source
func (h *Binlog) closeConsul() {
	if !h.enable {
		return
	}
	h.Delete(prefixKeepalive + h.sessionId)
	log.Debugf("current is leader %d", h.isLock)
	h.lock.Lock()
	l := h.isLock
	h.lock.Unlock()
	if l == 1 {
		log.Debugf("delete lock %s", h.LockKey)
		h.Unlock()
		h.Delete(h.LockKey)
	}
	h.Session.delete()
}

// write pos kv to consul
// use by src/library/binlog/handler.go SaveBinlogPostionCache
func (h *Binlog) Write(data []byte) bool {
	if !h.enable {
		return true
	}
	log.Debugf("write consul pos kv: %s, %v", posKey + h.LockKey, data)
	_, err := h.Kv.Put(&api.KVPair{Key: posKey + h.LockKey, Value: data}, nil)
	if err != nil {
		log.Errorf("write consul pos kv with error: %+v", err)
	}
	return nil == err
}

// lock if success, the current will be a leader
func (h *Binlog) Lock() bool {
	if !h.enable {
		return true
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		log.Errorf("error: %v", ErrorSessionEmpty)
		return false
	}
	//key string, value []byte, sessionID string
	p := &api.KVPair{Key: h.LockKey, Value: nil, Session: h.Session.ID}
	success, _, err := h.Kv.Acquire(p, nil)
	if err != nil {
		log.Errorf("lock error: %+v", err)
	}
	if success {
		h.lock.Lock()
		h.isLock = 1
		h.lock.Unlock()
	}
	return success
}

// unlock
func (h *Binlog) Unlock() bool {
	if !h.enable {
		return true
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		log.Errorf("error: %v", ErrorSessionEmpty)
		return false
	}
	p := &api.KVPair{Key: h.LockKey, Value: nil, Session: h.Session.ID}
	success, _, err := h.Kv.Release(p, nil)
	if err != nil {
		log.Errorf("lock error: %+v", err)
	}
	if success {
		h.lock.Lock()
		h.isLock = 0
		h.lock.Unlock()
	}
	return success
}

// delete a lock
func (h *Binlog) Delete(key string) error {
	if !h.enable {
		return nil
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		return nil
	}
	_, err := h.Kv.Delete(key, nil)
	if err == nil {
		h.lock.Lock()
		if key == h.LockKey {
			h.isLock = 0
		}
		h.lock.Unlock()
	}
	return err
}

