package agent

import (
	"time"
	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"fmt"
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
}


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

//
//// Next to return the updates
func (cw *ConsulWatcher) process() {
	// Nil cw.addrs means it is initial called
	// If get addrs, return to balancer
	// If no addrs, need to watch consul
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
			return
		}
	}
	for {
		// watch consul
		addrs, li, err := cw.queryConsul(&consul.QueryOptions{WaitIndex: cw.li})
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		fmt.Printf("service change: %+v, %+v\n", addrs, li)
		// generate updates
		//updates := genUpdates(cw.addrs, addrs)
		// update addrs & last index
		cw.addrs = addrs
		cw.li = li
		if len(addrs) > 0 {
			//如果发生改变的服务里面有leader，并且不是自己，则执行重新选leader
			for _, u:= range addrs {
				//u.Checks.AggregatedStatus()
				if len(u.Service.Tags) <= 0 {
					continue
				}
				if u.Service.Tags[0] != "isleader:true" {
					continue
				}
				if u.Service.Address != cw.serviceIp || u.Service.Port != cw.port {
					continue
				}
				for _, f := range cw.onChange {
					f()
				}
			}
			return
		}
	}
	// should never come here
	return
}
//
//// queryConsul is helper function to query consul
func (cw *ConsulWatcher) queryConsul(q *consul.QueryOptions) ([]*consul.ServiceEntry, uint64, error) {
	// query consul
	cs, meta, err := cw.health.Service(cw.target, "", true, q)
	if err != nil {
		return nil, 0, err
	}
	return cs, meta.LastIndex, nil
}