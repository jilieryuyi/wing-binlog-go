package agent

import (
	//"fmt"
	"time"
	consul "github.com/hashicorp/consul/api"
	//"google.golang.org/grpc/naming"
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
		fmt.Printf("query consul service\n")
		addrs, li, _ := cw.queryConsul(nil)
		fmt.Printf("service: %+v\n", addrs)
		// got addrs, return
		if len(addrs) > 0 {
			cw.addrs = addrs
			cw.li = li
			//当前自己的服务已经注册成功
			for _, a := range addrs {
				if a.Service.Address == cw.serviceIp && a.Service.Port == cw.port {
					for _, f := range cw.onChange {
						f()
					}
					break
				}
			}
			return// genUpdates([]string{}, addrs), nil
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
			return// updates, nil
		}
	}
	// should never come here
	return// []*naming.Update{}, nil
}
//
//// queryConsul is helper function to query consul
func (cw *ConsulWatcher) queryConsul(q *consul.QueryOptions) ([]*consul.ServiceEntry, uint64, error) {
	// query consul
	cs, meta, err := cw.health.Service(cw.target, "", true, q)
	if err != nil {
		return nil, 0, err
	}
	//addrs := make([]string, 0)
	//for _, s := range cs {
	//	fmt.Printf("service: %+v\n", *s)
	//	// addr should like: 127.0.0.1:8001
	//	addrs = append(addrs, fmt.Sprintf("%s:%d", s.Service.Address, s.Service.Port))
	//}
	return cs, meta.LastIndex, nil
}
// check update and delete
//func genUpdates(a, b []*consul.ServiceEntry) []*consul.ServiceEntry {
//	updates := make([]*consul.ServiceEntry, 0)
//	deleted := diff(a, b)
//	for _, addr := range deleted {
//		//update := &naming.Update{Op: naming.Delete, Addr: addr}
//		fmt.Printf("delete service: %s\n", addr)
//		updates = append(updates, addr)
//	}
//	added := diff(b, a)
//	for _, addr := range added {
//		fmt.Printf("new service: %s\n", addr)
//		//update := &naming.Update{Op: naming.Add, Addr: addr}
//		updates = append(updates, addr)
//	}
//	return updates
//}

// diff(a, b) = a - a(n)b
//func diff(a, b []*consul.ServiceEntry) []*consul.ServiceEntry {
//	d := make([]*consul.ServiceEntry, 0)
//	for _, va := range a {
//		found := false
//		vaAddress := fmt.Sprintf("%v:%v", va.Service.Address, va.Service.Port)
//		for _, vb := range b {
//			vbAddress := fmt.Sprintf("%v:%v", vb.Service.Address, vb.Service.Port)
//			if vaAddress == vbAddress {
//				found = true
//				break
//			}
//		}
//		if !found {
//			d = append(d, va)
//		}
//	}
//	return d
//}
//
