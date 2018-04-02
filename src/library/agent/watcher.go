package agent

import (
	//"fmt"
	//"time"
	consul "github.com/hashicorp/consul/api"
	//"google.golang.org/grpc/naming"
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
	addrs []string
	target string
	health *consul.Health
}

func newWatch(cc *consul.Client, serviceName string, health *consul.Health) *ConsulWatcher {
	return &ConsulWatcher{
		cc: cc,
		target: serviceName,
		health:health,
	}
}

// Close do nonthing
func (cw *ConsulWatcher) Close() {

}
//
//// Next to return the updates
//func (cw *ConsulWatcher) process() {
//	// Nil cw.addrs means it is initial called
//	// If get addrs, return to balancer
//	// If no addrs, need to watch consul
//	if cw.addrs == nil {
//		// must return addrs to balancer, use ticker to query consul till data gotten
//		fmt.Printf("query consul service\n")
//		addrs, li, _ := cw.queryConsul(nil)
//		fmt.Printf("service: %+v\n", addrs)
//		// got addrs, return
//		if len(addrs) != 0 {
//			cw.addrs = addrs
//			cw.li = li
//			return// genUpdates([]string{}, addrs), nil
//		}
//	}
//	for {
//		// watch consul
//		addrs, li, err := cw.queryConsul(&consul.QueryOptions{WaitIndex: cw.li})
//		if err != nil {
//			time.Sleep(1 * time.Second)
//			continue
//		}
//		fmt.Printf("service change: %+v, %+v\n", addrs, li)
//		// generate updates
//		updates := genUpdates(cw.addrs, addrs)
//		// update addrs & last index
//		cw.addrs = addrs
//		cw.li = li
//		if len(updates) != 0 {
//			return// updates, nil
//		}
//	}
//	// should never come here
//	return// []*naming.Update{}, nil
//}
//
//// queryConsul is helper function to query consul
//func (cw *ConsulWatcher) queryConsul(q *consul.QueryOptions) ([]string, uint64, error) {
//	// query consul
//	cs, meta, err := cw.health.Service(cw.target, "", true, q)
//	if err != nil {
//		return nil, 0, err
//	}
//	addrs := make([]string, 0)
//	for _, s := range cs {
//		fmt.Printf("service: %+v\n", *s)
//		// addr should like: 127.0.0.1:8001
//		addrs = append(addrs, fmt.Sprintf("%s:%d", s.Service.Address, s.Service.Port))
//	}
//	return addrs, meta.LastIndex, nil
//}
//
//// check update and delete
////func genUpdates(a, b []string) []*naming.Update {
////	updates := make([]*naming.Update, 0)
////	deleted := diff(a, b)
////	for _, addr := range deleted {
////		update := &naming.Update{Op: naming.Delete, Addr: addr}
////		fmt.Printf("delete service: %s\n", addr)
////		updates = append(updates, update)
////	}
////	added := diff(b, a)
////	for _, addr := range added {
////		fmt.Printf("new service: %s\n", addr)
////		update := &naming.Update{Op: naming.Add, Addr: addr}
////		updates = append(updates, update)
////	}
////	return updates
////}
////
////// diff(a, b) = a - a(n)b
////func diff(a, b []string) []string {
//	d := make([]string, 0)
//	for _, va := range a {
//		found := false
//		for _, vb := range b {
//			if va == vb {
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
