package agent

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	consul "github.com/hashicorp/consul/api"
	"sync"
	"strings"
	"os"
	"errors"
)

// 服务注册
const (
	Registered = 1 << iota
)
const (
	statusOnline    = "online"
	statusOffline   = "offline"
)
// cluster node(member)
type clusterMember struct {
	Hostname string
	IsLeader bool
	SessionId string
	Status string
	ServiceIp string
	Port int
}
type Service struct {
	ServiceName string //service name, like: service.add
	ServiceHost string //service host, like: 0.0.0.0, 127.0.0.1
	ServiceIp string // if ServiceHost is 0.0.0.0, ServiceIp must set,
	// like 127.0.0.1 or 192.168.9.12 or 114.55.56.168
	ServicePort int // service port, like: 9998
	Interval time.Duration // interval for update ttl
	Ttl int //check ttl
	ServiceID string //serviceID = fmt.Sprintf("%s-%s-%d", name, ip, port)
	client *consul.Client ///consul client
	agent *consul.Agent //consul agent
	status int // register status
	lock *sync.Mutex //sync lock
	leader bool
	lockKey string
	handler *consul.Session
	Kv *consul.KV
	lastSession string
	onleader []OnLeaderFunc
	health *consul.Health
}

type OnLeaderFunc func(bool)
type ServiceOption func(s *Service)

// set ttl
func Ttl(ttl int) ServiceOption {
	return func(s *Service){
		s.Ttl = ttl
	}
}

// set interval
func Interval(interval time.Duration) ServiceOption {
	return func(s *Service){
		s.Interval = interval
	}
}

// set service ip
func ServiceIp(serviceIp string) ServiceOption {
	return func(s *Service){
		s.ServiceIp = serviceIp
	}
}

// new a service
// name: service name
// host: service host like 0.0.0.0 or 127.0.0.1
// port: service port, like 9998
// consulAddress: consul service address, like 127.0.0.1:8500
// opts: ServiceOption, like ServiceIp("127.0.0.1")
// return new service pointer
func NewService(key string, name string, host string, port int,
	 c *consul.Client, opts ...ServiceOption) *Service {
	sev := &Service{
		lockKey:key,
		ServiceName:name,
		ServiceHost:host,
		ServicePort:port,
		Interval:time.Second * 10,
		Ttl: 15,
		status: 0,
		lock:new(sync.Mutex),
		leader:false,
		onleader:make([]OnLeaderFunc, 0),
	}
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(sev)
		}
	}
	sev.client = c
	sev.handler = c.Session()
	sev.Kv = c.KV()
	ip := host
	if ip == "0.0.0.0" {
		if sev.ServiceIp == "" {
			log.Panicf("please set consul service ip")
		}
		ip = sev.ServiceIp
	}
	sev.ServiceID = fmt.Sprintf("%s-%s-%d", name, ip, port)
	sev.agent = sev.client.Agent()
	go sev.updateTtl()
	sev.health = sev.client.Health()
	return sev
}

func (sev *Service) Deregister() error {
	err := sev.agent.ServiceDeregister(sev.ServiceID)
	if err != nil {
		log.Errorf("deregister service error: ", err.Error())
		return err
	}
	err = sev.agent.CheckDeregister(sev.ServiceID)
	if err != nil {
		log.Println("deregister check error: ", err.Error())
	}
	return err
}

func (sev *Service) updateTtl() {
	for {
		if sev.lastSession != "" {
			//log.Debugf("session renew")
			sev.handler.Renew(sev.lastSession, nil)
		}
		if sev.status & Registered <= 0 {
			time.Sleep(sev.Interval)
			continue
		}
		//log.Debugf("update ttl")
		err := sev.agent.UpdateTTL(sev.ServiceID, fmt.Sprintf("isleader:%v", sev.leader), "passing")
		if err != nil {
			log.Println("update ttl of service error: ", err.Error())
		}
		time.Sleep(sev.Interval)
	}
}

func (sev *Service) Register() error {
	sev.lock.Lock()
	if sev.status & Registered <= 0 {
		sev.status |= Registered
	}
	sev.lock.Unlock()
	//de-register if meet signhup
	// initial register service
	ip := sev.ServiceHost
	if ip == "0.0.0.0" {
		ip = sev.ServiceIp
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	regis := &consul.AgentServiceRegistration{
		ID:      sev.ServiceID,
		Name:    sev.ServiceName,
		Address: ip,
		Port:    sev.ServicePort,
		Tags:    []string{fmt.Sprintf("isleader:%v", sev.leader), sev.lockKey, hostname},
	}
	log.Debugf("service register")
	err = sev.agent.ServiceRegister(regis)
	if err != nil {
		return fmt.Errorf("initial register service '%s' host to consul error: %s", sev.ServiceName, err.Error())
	}
	// initial register service check
	check := consul.AgentServiceCheck{TTL: fmt.Sprintf("%ds", sev.Ttl), Status: "passing"}
	err = sev.agent.CheckRegister(&consul.AgentCheckRegistration{
		ID: sev.ServiceID,
		Name: sev.ServiceName,
		ServiceID: sev.ServiceID,
		AgentServiceCheck: check,
		})
	if err != nil {
		return fmt.Errorf("initial register service check to consul error: %s", err.Error())
	}
	return nil
}

func (sev *Service) Close() {
	sev.Deregister()
	if sev.leader {
		log.Debugf("=========>current is leader, try unlock")
		sev.Unlock()
		sev.Delete()
		sev.leader = false
	}
}

func (sev *Service) selectLeader() {
	log.Debugf("====start select leader====")
	leader, err := sev.Lock()
	if err != nil {
		log.Errorf("select leader with error: %v", err)
		return
	}
	sev.leader = leader
	//register for set tags isleader:true
	sev.Register()

	// 如果不是leader，然后检测当前的leader是否存在，如果不存在
	// 可以认为某些情况下发生了死锁，可以尝试强制解锁
	if !leader {
		_, _, err := sev.getLeader()
		if err == leaderNotFound{
			log.Debugf("check deadlock......please wait")
			time.Sleep(time.Second * 3)
		}
		_, _, err = sev.getLeader()
		//如果没有leader
		if err == leaderNotFound {
			log.Warnf("deadlock found, try to unlock")
			sev.Unlock()
			sev.Delete()
			log.Infof("select leader again")
			leader, err = sev.Lock()
			if err != nil {
				log.Errorf("select leader with error: %v", err)
				return
			}
			sev.leader = leader
			//register for set tags isleader:true
			sev.Register()
		}
	} else {
		// 如果选leader成功
		// 但是这个时候leader仍然不存在，可以认为网络问题造成注册服务失败
		// 这里尝试等待并重新注册
		for {
			_, _, err := sev.getLeader()
			if err == leaderNotFound {
				log.Warnf("leader not fund, try register")
				sev.Register()
				time.Sleep(time.Millisecond * 10)
				continue
			}
			break
		}
	}

	log.Debugf("select leader: %+v", leader)
	// 触发选leader成功相关事件回调
	if len(sev.onleader) > 0 {
		log.Debugf("leader on select fired")
		for _, f := range sev.onleader {
			f(leader)
		}
	}
}

func (sev *Service) createSession(/*timeOut int64*/) string {
	//if timeOut < 10 {
	//	timeOut = 10
	//}
	// 每个sev.Interval周期，session就会被刷新
	// 将session的ttl设置为这个周期的6倍
	// 一般情况下这个session永远不会被过期
	se := &consul.SessionEntry{
		Behavior : consul.SessionBehaviorDelete,
		TTL: fmt.Sprintf("%vs", int64(sev.Interval.Seconds() * 6)),
	}
	ID, _, err := sev.handler.Create(se, nil)
	if err != nil {
		log.Errorf("create session error: %+v", err)
		return ""
	}
	return ID
}

// lock if success, the current agent will be leader
func (sev *Service) Lock() (bool, error) {
	if sev.lastSession == "" {
		sev.lastSession = sev.createSession()
	}
	p := &consul.KVPair{Key: sev.lockKey, Value: nil, Session: sev.lastSession}
	success, _, err := sev.Kv.Acquire(p, nil)
	if err != nil {
		// try to create a new session
		log.Errorf("lock error: %+v", err)
		if strings.Contains(strings.ToLower(err.Error()), "session") {
			log.Errorf("try to create a new session")
			// session错误时，尝试重建session
			sev.lastSession = sev.createSession()
		}
		return false, err
	}
	return success, nil
}

// unlock
func (sev *Service) Unlock() (bool, error) {
	p := &consul.KVPair{Key: sev.lockKey, Value: nil, Session: sev.lastSession}
	success, _, err := sev.Kv.Release(p, nil)
	if err != nil {
		log.Errorf("unlock error: %+v", err)
		if strings.Contains(strings.ToLower(err.Error()), "session") {
			log.Errorf("try to create a new session")
			// session错误时，尝试重建session
			sev.lastSession = sev.createSession()
		}
		return false, err
	}
	return success, nil
}

// delete a lock
func (sev *Service) Delete() error {
	_, err := sev.Kv.Delete(sev.lockKey, nil)
	return err
}

func (sev *Service) getMembers() []*clusterMember {
	members, _, err := sev.health.Service(ServiceName, "", false, nil)
	if err != nil || members == nil {
		log.Errorf("get service list error: %+v", err)
		return nil
	}
	data := make([]*clusterMember, 0)
	for _, v := range members {
		log.Debugf("getMembers： %+v", *v.Service)
		// 这里的两个过滤，为了避免与其他服务冲突，只获取相同lockkey的服务，即 当前集群
		if len(v.Service.Tags) < 3 {
			continue
		}
		if v.Service.Tags[1] != sev.lockKey {
			continue
		}
		m := &clusterMember{}
		if v.Checks.AggregatedStatus() == "passing" {
			m.Status = statusOnline
			m.IsLeader  = v.Service.Tags[0] == "isleader:true"
		} else {
			m.Status = statusOffline
			m.IsLeader  = false//v.Service.Tags[0] == "isleader:true"
		}
		m.Hostname  = v.Service.Tags[2]
		m.SessionId = v.Service.ID//Tags[1]
		m.ServiceIp = v.Service.Address
		m.Port      = v.Service.Port
		data = append(data, m)
	}
	return data
}
var membersEmpty = errors.New("members is empty")
var leaderNotFound = errors.New("leader not found")
func (sev *Service) getLeader() (string, int, error) {
	members := sev.getMembers()
	if members == nil {
		return "", 0, membersEmpty
	}
	for _, v := range members {
		log.Debugf("getLeader: %+v", *v)
		if v.IsLeader {
			return v.ServiceIp, v.Port, nil
		}
	}
	return "", 0, leaderNotFound
}

func (sev *Service) ShowMembers() string {
	data := sev.getMembers()
	if data == nil {
		return ""
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = ""
	}
	res := fmt.Sprintf("current node: %s(%s:%d)\r\n", hostname, sev.ServiceIp, sev.ServicePort)
	res += fmt.Sprintf("cluster size: %d node(s)\r\n", len(data))
	res += fmt.Sprintf("======+=============================================+==========+===============\r\n")
	res += fmt.Sprintf("%-6s| %-43s | %-8s | %s\r\n", "index", "node", "role", "status")
	res += fmt.Sprintf("------+---------------------------------------------+----------+---------------\r\n")
	for i, member := range data {
		role := "follower"
		if member.IsLeader {
			role = "leader"
		}
		res += fmt.Sprintf("%-6d| %-43s | %-8s | %s\r\n", i, fmt.Sprintf("%s(%s:%d)", member.Hostname, member.ServiceIp, member.Port), role, member.Status)
	}
	res += fmt.Sprintf("------+---------------------------------------------+----------+---------------\r\n")
	return res
}
