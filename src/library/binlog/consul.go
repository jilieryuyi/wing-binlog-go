package binlog

import (
	log "github.com/sirupsen/logrus"
	"time"
	"os"
	"github.com/hashicorp/consul/api"
	"fmt"
	"strconv"
)

func (h *Binlog) consulInit() {
	var err error
	consulConfig, err := getConfig()

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
			h.Delete(LOCK)
		}
	}
	//超时检测，即检测leader是否挂了，如果挂了，要重新选一个leader
	//如果当前不是leader，重新选leader。leader不需要check
	//如果被选为leader，则还需要执行一个onLeader回调
	go h.checkAlive()
	//////还需要一个keepalive
	go h.keepalive()
	////还需要一个检测pos变化回调，即如果不是leader，要及时更新来自leader的pos变化
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
		if v != nil && v.Session == h.sessionId {
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
	//name := hostname + h.sessionId
	t := time.Now().Unix()
	service := &api.AgentServiceRegistration{
		ID:                h.sessionId,
		Name:              h.sessionId,
		Tags:              []string{fmt.Sprintf("%d", h.isLock), h.sessionId, fmt.Sprintf("%d", t), hostname},
		Port:              h.ServicePort,
		Address:           h.ServiceIp,
		EnableTagOverride: false,
		Check:             nil,
		Checks:            nil,
	}
	log.Debugf("register service %+v", *service)
	err = h.agent.ServiceRegister(service)
	if err != nil {
		log.Errorf("register service with error: %+v", err)
	}
}

// 服务发现，获取服务列表
func (h *Binlog) GetServices() map[string]*api.AgentService {
	if !h.enable {
		return nil
	}
	//1516574111-0hWR-E6IN-lrsO: {
	// ID:1516574111-0hWR-E6IN-lrsO
	// Service:yuyideMacBook-Pro.local1516574111-0hWR-E6IN-lrsO
	// Tags:[ 1516574111-0hWR-E6IN-lrsO /7tZ yuyideMacBook-Pro.local]
	// Port:9998 Address:127.0.0.1
	// EnableTagOverride:false
	// CreateIndex:0
	// ModifyIndex:0
	// }
	ser, err := h.agent.Services()
	if err != nil {
		log.Errorf("get service list error: %+v", err)
		return nil
	}
	return ser
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
	members := h.GetServices()
	if members == nil {
		return nil
	}
	m := make([]*ClusterMember, len(members))
	var i = 0
	for _, v := range members {
		m[i] = &ClusterMember{}
		t, _:= strconv.ParseInt(v.Tags[2], 10, 64)
		m[i].Status = statusOnline
		if time.Now().Unix() - t > serviceKeepaliveTimeout {
			m[i].Status = statusOffline
		}
		m[i].IsLeader  = v.Tags[0] == "1"
		m[i].Hostname  = v.Tags[3]
		m[i].Session   = v.Tags[1]
		m[i].ServiceIp = v.Address
		m[i].Port      = v.Port
		log.Debugf("member=>%+v", *m[i])
		i++
	}
	return m
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
		services := h.GetServices()
		if services == nil {
			time.Sleep(time.Second * checkAliveInterval)
			continue
		}
		for _, v := range services {
			isLock := v.Tags[0] == "1"
			t, _ := strconv.ParseInt(v.Tags[2], 10, 64)
			if time.Now().Unix()-t > serviceKeepaliveTimeout {
				log.Warnf("%s is timeout, will be deregister", v.ID)
				h.agent.ServiceDeregister(v.ID)
				// if is leader, try delete lock and reselect a new leader
				if isLock {
					h.Delete(LOCK)
					if h.Lock() {
						log.Debugf("current is the new leader")
						//if h.onLeaderCallback != nil {
						//	h.onLeaderCallback()
						//}
						h.onNewLeader()
						//h.ctx.NewLeader <- struct{}{}
					}
				}
			}
		}
		time.Sleep(time.Second * checkAliveInterval)
	}
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
		_, meta, err := h.Kv.Get(posKey, nil)
		if err != nil {
			log.Errorf("watch pos change with error：%#v", err)
			time.Sleep(time.Second)
			continue
		}
		if meta == nil {
			time.Sleep(time.Second)
			continue
		}
		v, _, err := h.Kv.Get(posKey, &api.QueryOptions{
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
		log.Debugf("delete lock %s", LOCK)
		h.Unlock()
		h.Delete(LOCK)
	}
	h.Session.delete()
}

// write pos kv to consul
// use by src/library/binlog/handler.go SaveBinlogPostionCache
func (h *Binlog) Write(data []byte) bool {
	if !h.enable {
		return true
	}
	log.Debugf("write consul pos kv: %s, %v", posKey, data)
	_, err := h.Kv.Put(&api.KVPair{Key: posKey, Value: data}, nil)
	if err != nil {
		log.Errorf("write consul pos kv with error: %+v", err)
	}
	return nil == err
}
