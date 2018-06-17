package subscribe

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"time"
	consul "github.com/hashicorp/consul/api"
	"sync"
	"os"
	"net"
	"sync/atomic"
	"encoding/binary"
)

// 服务注册
const (
	Registered = 1 << iota
)
const (
	statusOnline    = "online"
	statusOffline   = "offline"
)
// todo 这里还需要一个操作，就是，客户端接入或者断开的时候，触发更新服务的属性
// 即将当前连接数接入到consul服务，客户端做服务发现的时候，自动优先连接连接数最少的

type Service struct {
	ServiceName string 		//service name, like: service.add
	ServiceHost string 		//service host, like: 0.0.0.0, 127.0.0.1
	ServiceIp string   		// if ServiceHost is 0.0.0.0, ServiceIp must set,
					   		// like 127.0.0.1 or 192.168.9.12 or 114.55.56.168
	ServicePort int    		// service port, like: 9998
	Interval time.Duration 	// interval for update ttl
	Ttl int 		   		//check ttl
	ServiceID string  		//serviceID = fmt.Sprintf("%s-%s-%d", name, ip, port)
	client *consul.Client 	//consul client
	agent *consul.Agent 	//consul agent
	status int 				// register status
	lock *sync.Mutex 		//sync lock
	handler *consul.Session
	Kv *consul.KV
	onleader []OnLeaderFunc
	health *consul.Health
	connects int64
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
func NewService(
	host string,
	port int,
	consulAddress string,
	opts ...ServiceOption) *Service {
	sev := &Service{
		ServiceName:ServiceName,
		ServiceHost:host,
		ServicePort:port,
		Interval:time.Second * 10,
		Ttl: 15,
		status: 0,
		lock:new(sync.Mutex),
		onleader:make([]OnLeaderFunc, 0),
		connects:int64(0),
	}
	for _, opt := range opts {
		opt(sev)
	}
	conf    := &consul.Config{Scheme: "http", Address: consulAddress}
	c, err  := consul.NewClient(conf)
	if err != nil {
		log.Printf("%v", err)
	}
	//c :=//*consul.Client
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
	sev.ServiceID = fmt.Sprintf("%s-%s-%d", ServiceName, ip, port)
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
	ip := sev.ServiceHost
	if ip == "0.0.0.0" && sev.ServiceIp != "" {
		ip = sev.ServiceIp
	}
	key := fmt.Sprintf("connects/%v/%v", ip, sev.ServicePort)
	se := &consul.SessionEntry{
		Behavior : consul.SessionBehaviorDelete,
		TTL: fmt.Sprintf("%vs", sev.Interval.Seconds() * 3),
	}
	session, _, err := sev.handler.Create(se, nil)
	if err != nil {
		log.Panicf("%+v", err)
	}
	for {
		_, _, err = sev.handler.Renew(session, nil)
		if err != nil {
			log.Errorf("%+v", err)
		}
		count := atomic.LoadInt64(&sev.connects)
		var data = make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(count))
		p := &consul.KVPair{
			Key: key,
			Value: data,//[]byte(fmt.Sprintf("%v", count)),
			Session: session,
		}
		_, err = sev.Kv.Put(p, nil)
		if err != nil {
			log.Errorf("%+v", err)
		}
		if sev.status & Registered <= 0 {
			time.Sleep(sev.Interval)
			continue
		}
		err = sev.agent.UpdateTTL(sev.ServiceID, "", "passing")
		if err != nil {
			log.Println("update ttl of service error: ", err.Error())
		}
		time.Sleep(sev.Interval)
	}
}

func (sev *Service) newConnect(conn *net.Conn) {
	log.Debugf("##############service new connect##############")
	atomic.AddInt64(&sev.connects, 1)
}
func (sev *Service) disconnect(conn *net.Conn) {
	log.Debugf("##############service new disconnect##############")
	atomic.AddInt64(&sev.connects, -1)
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
	if ip == "0.0.0.0" && sev.ServiceIp != "" {
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
		Tags:    []string{hostname},
	}
	log.Debugf("subscribe service register: %+v", *regis)
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
}
