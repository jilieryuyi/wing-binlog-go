package agent

import (
	"time"
	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

//监听服务变化
//如果leader服务变为不可用状态
//则重新选举leader

// ConsulWatcher is the implementation of grpc.naming.Watcher
type ConsulWatcher struct {
	// cc: Consul Client
	cc *consul.Client
	// LastIndex to watch consul
	li uint64
	// addrs is the service address cache
	// before check: every value shoud be 1
	// after check: 1 - deleted  2 - nothing  3 - new added
	addrs []*consul.ServiceEntry//[]string
	target string
	health *consul.Health
	onChange []onchangeFunc
	serviceIp string
	port int
	unlock unlockFunc
}

type unlockFunc func() (bool, error)
type watchOption func(w *ConsulWatcher)
type onchangeFunc func()//ip string, port int, isLeader bool)

func newWatch(cc *consul.Client, serviceName string,
	health *consul.Health, serviceIp string, port int,
		opts ...watchOption) *ConsulWatcher {
	w := &ConsulWatcher{
		cc: cc,
		target: serviceName,
		health:health,
		serviceIp:serviceIp,
		port:port,
	}
	if len(opts) > 0 {
		for _, f := range opts {
			f(w)
		}
	}
	return  w
}

func onWatch(f onchangeFunc) watchOption {
	return func(w *ConsulWatcher) {
		w.onChange = append(w.onChange, f)
	}
}

func unlock(f unlockFunc) watchOption {
	return func(w *ConsulWatcher) {
		w.unlock = f
	}
}

//
//// Next to return the updates
func (cw *ConsulWatcher) process() {
	// Nil cw.addrs means it is initial called
	// If get addrs, return to balancer
	// If no addrs, need to watch consul
	for {
		if cw.addrs == nil {
			// must return addrs to balancer, use ticker to query consul till data gotten
			log.Debugf("query consul service")
			addrs, li, _ := cw.queryConsul(nil)
			log.Debugf("service: %+v", addrs)
			// got addrs, return
			if len(addrs) > 0 {
				cw.addrs = addrs
				cw.li = li
				//当前自己的服务已经注册成功
				for _, a := range addrs {
					log.Debugf("addr: %+v", *a)
					if a.Service.Address == cw.serviceIp && a.Service.Port == cw.port {
						log.Debugf("fired cw.onChange")
						for _, f := range cw.onChange {
							f()
						}
						break
					}
				}
			}
			continue
		}
		for {
			// watch consul
			addrs, li, err := cw.queryConsul(&consul.QueryOptions{WaitIndex: cw.li})
			if err != nil {
				log.Errorf("============>cw.queryConsul error: %+v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			log.Debugf("============>service change: %+v, %+v", addrs, li)
			// generate updates
			//updates := genUpdates(cw.addrs, addrs)
			// update addrs & last index

			if len(addrs) > 0 {
				deleted := getDelete(cw.addrs, addrs)
				//for _, addr := range deleted {
				//	//update := &naming.Update{Op: naming.Delete, Addr: addr}
				//	log.Debugf("delete or status change service: %s\n", addr)
				//	updates = append(updates, addr)
				//}
				//updates := genUpdates(cw.addrs, addrs)
				//如果发生改变的服务里面有leader，并且不是自己，则执行重新选leader
				for _, u := range deleted {
					for {
						log.Debugf("====>delete service: %+v", *u.Service)
						//u.Checks.AggregatedStatus()
						// error data
						if len(u.Service.Tags) <= 0 {
							log.Errorf("tag is error")
							break
						}
						// if not leader
						// 发生改变的不是leader，无须理会，只有leader发生改变才需要执行重选leader
						if u.Service.Tags[0] != "isleader:true" {
							log.Errorf("is not leader")
							break
						}
						// if leader runs ok
						// 如果是leader，并且leader正常运行，也无需例会
						//if u.Service.Tags[0] == "isleader:true" && u.Checks.AggregatedStatus() == "passing" {
						//	log.Errorf("leader is running")
						//	break
						//}
						// check is self
						// 如果是当前节点，也无需处理
						if u.Service.Address == cw.serviceIp && u.Service.Port == cw.port {
							log.Warnf("is current node")
							break
						}
						// try to unlock
						for i:=0;i<3;i++ {
							s, err := cw.unlock()
							if s {
								break
							}
							if err != nil {
								log.Errorf("unlock error: %+v", err)
							}
						}
						log.Debugf("============>fired cw.onChange<====")
						for _, f := range cw.onChange {
							f()
						}
						break
					}
				}

				changed := getChange(cw.addrs, addrs)
				for _, u := range changed {
					for {
						log.Debugf("====>status change service: %+v", *u.Service)
						//u.Checks.AggregatedStatus()
						// error data
						if len(u.Service.Tags) <= 0 {
							log.Errorf("tag is error")
							break
						}
						// if not leader
						// 发生改变的不是leader，无须理会，只有leader发生改变才需要执行重选leader
						if u.Service.Tags[0] != "isleader:true" {
							log.Errorf("is not leader")
							break
						}
						// if leader runs ok
						// 如果是leader，并且leader正常运行，也无需例会
						if u.Service.Tags[0] == "isleader:true" && u.Checks.AggregatedStatus() == "passing" {
							log.Errorf("leader is running")
							break
						}
						// check is self
						// 如果是当前节点，也无需处理
						if u.Service.Address == cw.serviceIp && u.Service.Port == cw.port {
							log.Warnf("is current node")
							break
						}
						log.Debugf("============>fired cw.onChange<====")
						// try to unlock
						for i:=0;i<3;i++ {
							s, err := cw.unlock()
							if s {
								break
							}
							if err != nil {
								log.Errorf("unlock error: %+v", err)
							}
						}
						for _, f := range cw.onChange {
							f()
						}
						break
					}
				}
			}
			cw.addrs = addrs
			cw.li = li
			//time.Sleep(3 * time.Second)
		}
	}
}
//
//// queryConsul is helper function to query consul
func (cw *ConsulWatcher) queryConsul(q *consul.QueryOptions) ([]*consul.ServiceEntry, uint64, error) {
	// query consul
	cs, meta, err := cw.health.Service(cw.target, "", false, q)
	if err != nil {
		return nil, 0, err
	}
	return cs, meta.LastIndex, nil
}


// check update and delete
//func genUpdates(a, b []*consul.ServiceEntry) []*consul.ServiceEntry {
//	updates := make([]*consul.ServiceEntry, 0)
//	deleted := diff(a, b)
//	for _, addr := range deleted {
//		//update := &naming.Update{Op: naming.Delete, Addr: addr}
//		log.Debugf("delete or status change service: %s\n", addr)
//		updates = append(updates, addr)
//	}
//
//	added := diff(b, a)
//	for _, addr := range added {
//		log.Debugf("new or status change service: %s\n", addr)
//		//update := &naming.Update{Op: naming.Add, Addr: addr}
//		updates = append(updates, addr)
//	}
//
//	return updates
//}

// diff(a, b) = a - a(n)b
func getDelete(a, b []*consul.ServiceEntry) ([]*consul.ServiceEntry) {
	d := make([]*consul.ServiceEntry, 0)
	//exists := make([]*consul.ServiceEntry, 0)
	for _, va := range a {
		found := false
		//statusChange := false
		for _, vb := range b {
			if va.Service.ID == vb.Service.ID {
				found = true
				// 如果已存在，对比一下状态是否已发生改变
				// 如果已经改变追加到d里面返回
				if va.Checks.AggregatedStatus() != vb.Checks.AggregatedStatus() {
					log.Debugf("status change: %+v", )
					// 如果已存在，对比一下状态是否已发生改变
					// 如果已经改变追加到d里面返回
					//statusChange = true
					//d = append(d, vb)
				}
				break
			}
		}
		if !found {
			d = append(d, va)
		}
		//if found && statusChange {
		//	// 如果已存在，对比一下状态是否已发生改变
		//	// 如果已经改变追加到d里面返回
		//	d = append(d, va)
		//}
	}
	return d
}

func getChange(a, b []*consul.ServiceEntry) ([]*consul.ServiceEntry) {
	d := make([]*consul.ServiceEntry, 0)
	//exists := make([]*consul.ServiceEntry, 0)
	for _, va := range a {
		//found := false
		//statusChange := false
		for _, vb := range b {
			if va.Service.ID == vb.Service.ID && va.Checks.AggregatedStatus() != vb.Checks.AggregatedStatus() {
				//found = true
				// 如果已存在，对比一下状态是否已发生改变
				// 如果已经改变追加到d里面返回
				log.Debugf("status change: %+v", )
				// 如果已存在，对比一下状态是否已发生改变
				// 如果已经改变追加到d里面返回
				//statusChange = true
				d = append(d, vb)
				break
			}
		}
		//if !found {
		//	d = append(d, va)
		//}
		//if found && statusChange {
		//	// 如果已存在，对比一下状态是否已发生改变
		//	// 如果已经改变追加到d里面返回
		//	d = append(d, va)
		//}
	}
	return d
}
