package consul

import (
	"time"
	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

const (
	EventStatusChange = iota+1
	EventDelete
	EventAdd
)

// WatchService is the implementation of grpc.naming.Watcher
type WatchService struct {
	// LastIndex to watch consul
	li uint64
	// addrs is the service address cache
	// before check: every value shoud be 1
	// after check: 1 - deleted  2 - nothing  3 - new added
	addrs []*consul.ServiceEntry//[]string
	target string
	health *consul.Health
	// unlock api, come form service.go(Unlock)
	//unlock unlockFunc
}

//type unlockFunc func() (bool, error)
type WatchOption func(w *WatchService)

// watch service change
func NewWatchService(health *consul.Health, serviceName string, opts ...WatchOption, ) *WatchService {
	w := &WatchService{
		target:    serviceName,
		health:    health,
	}
	for _, f := range opts {
		f(w)
	}
	return  w
}

// watch service delete and change
func (cw *WatchService) Watch(watch func(int, *ServiceMember)) {
	// Nil cw.addrs means it is initial called
	// If get addrs, return to balancer
	// If no addrs, need to watch consul
	go func() {
	for {
		if cw.addrs == nil {
			// must return addrs to balancer, use ticker to query consul till data gotten
			log.Infof("query consul service")
			addrs, li, err := cw.queryConsul(nil)
			log.Infof("service: %v, %+v", li,addrs)
			// got addrs, return
			if err == nil {
				cw.addrs = addrs
				cw.li    = li
				//当前自己的服务已经注册成功
				for _, a := range addrs {
					log.Infof("addr: %+v", *a)
				}
			} else {
				time.Sleep(time.Second)
			}
			continue
		}
		for {
			// watch consul
			addrs, li, err := cw.queryConsul(&consul.QueryOptions{WaitIndex: cw.li})
			log.Infof("watch return %v services", len(addrs))
			if err != nil {
				log.Errorf("============>cw.queryConsul error: %+v", err)
				time.Sleep(1 * time.Second)
				continue
			}



			// 这里意欲监听leader的变化
			// 比如删除了leader，需要触发重选leader
			// 比如leader状态发生了变化，编程了不可用状态，这个时候也需要触发重选leader
			if addrs == nil {
				log.Warnf("watch services return nil")
				addrs = make([]*consul.ServiceEntry, 0)
			} else {
				for _, a := range addrs {
					log.Infof("watch addr: %+v\r\n%+v", *a, *a.Service)
				}
			}
			cw.dialAdd(watch, addrs)
			cw.dialDelete(watch, addrs)
			cw.dialChange(watch, addrs)

			cw.addrs = addrs
			cw.li = li
		}
	}
	}()
}

func (cw *WatchService) dialChange(watch func(int, *ServiceMember), addrs []*consul.ServiceEntry) {
	changed := getChange(cw.addrs, addrs)
	for _, u := range changed {
		status := statusOffline
		if u.Checks.AggregatedStatus()  == "passing" {
			status = statusOnline
		}
		leader := false
		if len(u.Service.Tags) > 0 && u.Service.Tags[0] == "isleader:true" {
			leader = true
		}
		watch(EventStatusChange, &ServiceMember{
			IsLeader:  leader,
			ServiceID: u.Service.ID,
			Status:    status,
			ServiceIp: u.Service.Address,
			Port:      u.Service.Port,
		})
	}
}

func (cw *WatchService) dialAdd(watch func(int, *ServiceMember), addrs []*consul.ServiceEntry) {
	deleted := getDelete(addrs, cw.addrs)
	//如果发生改变的服务里面有leader，并且不是自己，则执行重新选leader
	for _, u := range deleted {
		status := statusOffline
		if u.Checks.AggregatedStatus()  == "passing" {
			status = statusOnline
		}
		leader := false
		if len(u.Service.Tags) > 0 && u.Service.Tags[0] == "isleader:true" {
			leader = true
		}
		watch(EventAdd, &ServiceMember{
			IsLeader:  leader,
			ServiceID: u.Service.ID,
			Status:    status,
			ServiceIp: u.Service.Address,
			Port:      u.Service.Port,
		})
	}
}

func (cw *WatchService) dialDelete(watch func(int, *ServiceMember), addrs []*consul.ServiceEntry) {
	deleted := getDelete(cw.addrs, addrs)
	//如果发生改变的服务里面有leader，并且不是自己，则执行重新选leader
	for _, u := range deleted {
		status := statusOffline
		if u.Checks.AggregatedStatus()  == "passing" {
			status = statusOnline
		}
		leader := false
		if len(u.Service.Tags) > 0 && u.Service.Tags[0] == "isleader:true" {
			leader = true
		}
		watch(EventDelete, &ServiceMember{
			IsLeader:  leader,
			ServiceID: u.Service.ID,
			Status:    status,
			ServiceIp: u.Service.Address,
			Port:      u.Service.Port,
		})
	}
}

// queryConsul is helper function to query consul
func (cw *WatchService) queryConsul(q *consul.QueryOptions) ([]*consul.ServiceEntry, uint64, error) {
	// query consul
	cs, meta, err := cw.health.Service(cw.target, "", false, q)
	if err != nil {
		return nil, 0, err
	}
	return cs, meta.LastIndex, nil
}

// diff(a, b) = a - a(n)b
func getDelete(a, b []*consul.ServiceEntry) ([]*consul.ServiceEntry) {
	d := make([]*consul.ServiceEntry, 0)
	for _, va := range a {
		found := false
		for _, vb := range b {
			if va.Service.ID == vb.Service.ID {
				found = true
				break
			}
		}
		if !found {
			d = append(d, va)
		}
	}
	return d
}

func getChange(a, b []*consul.ServiceEntry) ([]*consul.ServiceEntry) {
	d := make([]*consul.ServiceEntry, 0)
	for _, va := range a {
		for _, vb := range b {
			if va.Service.ID == vb.Service.ID && va.Checks.AggregatedStatus() != vb.Checks.AggregatedStatus() {
				// 如果已存在，对比一下状态是否已发生改变
				// 如果已经改变追加到d里面返回
				log.Debugf("status change: %v!=%v", va.Checks.AggregatedStatus() , vb.Checks.AggregatedStatus())
				// 如果已存在，对比一下状态是否已发生改变
				// 如果已经改变追加到d里面返回
				//statusChange = true
				d = append(d, vb)
				break
			}
		}
	}
	return d
}
