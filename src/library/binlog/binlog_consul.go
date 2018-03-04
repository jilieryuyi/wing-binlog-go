package binlog

import (
	log "github.com/sirupsen/logrus"
	"time"
	"os"
	"github.com/hashicorp/consul/api"
	"fmt"
	"strconv"
	"net"
	"strings"
	"library/app"
)

func (h *Binlog) consulInit() {
	var err error
	//consulConfig, err := getConfig()
	h.LockKey     = h.ctx.ClusterConfig.Lock
	h.Address     = h.ctx.ClusterConfig.Consul.Address
	h.sessionId   = app.GetKey(app.CachePath + "/session")
	if h.ctx.ClusterConfig.Enable {
		h.status ^= disableConsul
		h.status |= enableConsul
	}
	ConsulConfig := api.DefaultConfig()
	ConsulConfig.Address = h.Address
	h.Client, err = api.NewClient(ConsulConfig)
	if err != nil {
		log.Panicf("create consul session with error: %+v", err)
	}
	h.Session = &Session {
		Address : h.Address,
		ID      : "",
		handler : h.Client.Session(),
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
	go h.checkAlive()
	go h.keepalive()
}

func (h *Binlog) getService() *ClusterMember {
	if h.status & disableConsul > 0 {
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
	if h.status & disableConsul > 0 {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	t := time.Now().Unix()
	isLeader := 0
	if h.status & consulIsLeader > 0 {
		isLeader = 1
	}
	service := &api.AgentServiceRegistration{
		ID:                h.sessionId,
		Name:              h.LockKey,
		Tags:              []string{fmt.Sprintf("%d", isLeader), h.sessionId, fmt.Sprintf("%d", t), hostname, h.LockKey},
		Port:              h.ServicePort,
		Address:           h.ServiceIp,
		EnableTagOverride: false,
		Check:             nil,
		Checks:            nil,
	}
	//log.Debugf("register service: %+v", *service)
	err = h.agent.ServiceRegister(service)
	if err != nil {
		log.Errorf("register service with error: %+v", err)
	}
}

func (h *Binlog) GetCurrent() (string, int) {
	return h.ServiceIp, h.ServicePort
}

// keepalive
func (h *Binlog) keepalive() {
	if h.status & disableConsul > 0 {
		return
	}
	for {
		select {
		case <- h.ctx.Ctx.Done():
			return
			default:
		}
		h.Session.renew()
		h.registerService()
		time.Sleep(time.Second * keepaliveInterval)
	}
}

func (h *Binlog) ShowMembers() string {
	if h.status & disableConsul > 0 {
		return ""
	}
	members := h.GetMembers()
	currentIp, currentPort := h.GetCurrent()
	if members != nil {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = ""
		}
		l := len(members)
		res := fmt.Sprintf("current node: %s(%s:%d)\r\n", hostname, currentIp, currentPort)
		res += fmt.Sprintf("cluster size: %d node(s)\r\n", l)
		res += fmt.Sprintf("======+=============================================+==========+===============\r\n")
		res += fmt.Sprintf("%-6s| %-43s | %-8s | %s\r\n", "index", "node", "role", "status")
		res += fmt.Sprintf("------+---------------------------------------------+----------+---------------\r\n")
		for i, member := range members {
			role := "follower"
			if member.IsLeader {
				role = "leader"
			}
			res += fmt.Sprintf("%-6d| %-43s | %-8s | %s\r\n", i, fmt.Sprintf("%s(%s:%d)", member.Hostname, member.ServiceIp, member.Port), role, member.Status)
		}
		res += fmt.Sprintf("------+---------------------------------------------+----------+---------------\r\n")
		return res
	} else {
		return ""
	}
}

// get all members nodes
func (h *Binlog) GetMembers() []*ClusterMember {
	if h.status & disableConsul > 0 {
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
		if (time.Now().Unix() - t > serviceKeepaliveTimeout) && !h.alive(m.ServiceIp, m.Port) {
			m.Status = statusOffline
			log.Debugf("now: %d, t:%d, diff: %d", time.Now().Unix(), t, time.Now().Unix() - t)
		}
		m.IsLeader  = v.Tags[0] == "1"
		m.Hostname  = v.Tags[3]
		m.SessionId = v.Tags[1]
		m.ServiceIp = v.Address
		m.Port      = v.Port
		data = append(data, m)
	}
	return data
}

// check service is alive
// if leader is not alive, try to select a new one
func (h *Binlog) checkAlive() {
	if h.status & disableConsul > 0 {
		return
	}
	time.Sleep(30)
	for {
		select {
		case <- h.ctx.Ctx.Done():
			return
		default:
		}
		members := h.GetMembers()
		if members == nil {
			time.Sleep(time.Second * checkAliveInterval)
			continue
		}
		leaderCount := 0
		for _, v := range members {
			if v.IsLeader && v.Status != statusOffline {
				leaderCount++
			}
			if v.SessionId == h.sessionId {
				continue
			}
			if v.Status == statusOffline {
				log.Warnf("%s is timeout", v.SessionId)
				h.agent.ServiceDeregister(v.SessionId)
				// if is leader, try delete lock and reselect a new leader
				// if is leader, ping, check leader is alive again
				if v.IsLeader {
					h.Delete(h.LockKey)
				}
			}
		}
		if leaderCount == 0 && len(members) > 0 {
			//log.Warnf("no leader is running")
			//for _, v := range members {
			//	log.Debugf("member: %+v", *v)
			//}
			// current not leader
			if h.status & consulIsFollower > 0 {
				log.Warnf("current is not leader, will unlock")
				h.Delete(h.LockKey)
			}
		}
		if leaderCount > 1 {
			log.Warnf("%d leaders is running", leaderCount)
			h.Delete(h.LockKey)
		}
		time.Sleep(time.Second * checkAliveInterval)
	}
}

func (h *Binlog) alive(ip string, port int) bool {
	if ip == "" || port <= 0 {
		return true
	}
	log.Debugf("check alive %s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", ip, port), time.Second * 3)
	if err != nil || conn == nil {
		log.Debugf("is not alive: %v", err)
		return false
	}
	conn.SetWriteDeadline(time.Now().Add(time.Second * 3))
	conn.Write(pingData)
	var buf = make([]byte, 64)
	n, err := conn.Read(buf)
	log.Debugf("===>%v, %v", n, err)
	conn.Close()
	return true
}

// get leader service ip and port
// if not found or some error happened
// return empty string and 0
func (h *Binlog) GetLeader() (string, int) {
	if h.status & disableConsul > 0 {
		return "", 0
	}
	members := h.GetMembers()
	if members == nil || len(members) == 0 {
		return "", 0
	}
	for _, v := range members {
		if v.IsLeader && v.Status == statusOnline {
			return v.ServiceIp, v.Port
			break
		}
	}
	return "", 0
}

// if app is close, it will be call for clear some source
func (h *Binlog) closeConsul() {
	if h.status & disableConsul > 0  {
		return
	}
	h.Delete(prefixKeepalive + h.sessionId)
	if h.status & consulIsLeader > 0 {
		log.Debugf("delete lock %s", h.LockKey)
		h.Unlock()
		h.Delete(h.LockKey)
	}
	h.Session.delete()
}

// lock if success, the current will be a leader
func (h *Binlog) Lock() (bool, error) {
	if h.status & disableConsul > 0 {
		return true, nil
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		return false, sessionEmpty
	}
	//key string, value []byte, sessionID string
	p := &api.KVPair{Key: h.LockKey, Value: nil, Session: h.Session.ID}
	success, _, err := h.Kv.Acquire(p, nil)
	if err != nil {
		// try to create a new session
		log.Errorf("lock error: %+v", err)
		if strings.Contains(strings.ToLower(err.Error()), "session") {
			log.Errorf("try to create a new session")
			h.Session.create()
		}
		return false, err
	}
	if success && h.status & consulIsFollower > 0 {
		h.status ^= consulIsFollower
		h.status |= consulIsLeader
		//log.Debugf("===============>current node is leader")
	}
	return success, nil
}

// unlock
func (h *Binlog) Unlock() (bool, error) {
	if h.status & disableConsul > 0 {
		return true, nil
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		return false, sessionEmpty
	}
	p := &api.KVPair{Key: h.LockKey, Value: nil, Session: h.Session.ID}
	success, _, err := h.Kv.Release(p, nil)
	if err != nil {
		log.Errorf("unlock error: %+v", err)
		if strings.Contains(strings.ToLower(err.Error()), "session") {
			log.Errorf("try to create a new session")
			h.Session.create()
		}
		return false, err
	}
	if success && h.status & consulIsLeader > 0 {
		h.status ^= consulIsLeader
		h.status |= consulIsFollower
		//log.Debugf("===============>current node is follower")
	}
	return success, nil
}

// delete a lock
func (h *Binlog) Delete(key string) error {
	if h.status & disableConsul > 0 {
		return nil
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		return nil
	}
	_, err := h.Kv.Delete(key, nil)
	if err == nil && key == h.LockKey && h.status & consulIsLeader > 0 {
		h.status ^= consulIsLeader
		h.status |= consulIsFollower
	}
	return err
}

