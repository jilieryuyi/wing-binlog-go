package client

import (
	"time"
	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"fmt"
	"encoding/binary"
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
	health *consul.Health
	// leader change callback
	onChange []onChangeFunc
}

const (
	EV_ADD = 1

	EV_DELETE = 2 // need to delete service
	EV_CHANGE = 3 // status change to not reachable, need to delete service
)

type watchOption func(w *ConsulWatcher)
type onChangeFunc func(ip string, port int, event int)//ip string, port int, isLeader bool)

func newWatch(
	consulAddress string,
	opts ...watchOption) *ConsulWatcher {
	conf    := &consul.Config{Scheme: "http", Address: consulAddress}
	cc, err  := consul.NewClient(conf)
	if err != nil {
		log.Printf("%v", err)
	}
	w := &ConsulWatcher{
		cc: cc,
		health:cc.Health(),
	}
	for _, f := range opts {
		f(w)
	}
	go w.process()
	return  w
}

func onWatch(f onChangeFunc) watchOption {
	return func(w *ConsulWatcher) {
		w.onChange = append(w.onChange, f)
	}
}

// watch service delete and change
func (cw *ConsulWatcher) process() {
	// Nil cw.addrs means it is initial called
	// If get addrs, return to balancer
	// If no addrs, need to watch consul
		
		for {
			// watch consul
			addrs, li, err := cw.queryConsul(&consul.QueryOptions{WaitIndex: cw.li})
			if err != nil {
				log.Errorf("============>cw.queryConsul error: %+v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			log.Debugf("============>service change: %+v, %+v", addrs, li)
			if addrs == nil {
				log.Warnf("watch consul services return nil")
				addrs = make([]*consul.ServiceEntry, 0)
			}
			
			cw.dialDelete(addrs)
			cw.dialChande(addrs)
			cw.dialAdd(addrs)
			
			cw.addrs = addrs
			cw.li = li
		}
			
		
}

func (cw *ConsulWatcher) dialChande(addrs []*consul.ServiceEntry) {
	changed := getChange(cw.addrs, addrs)
	for _, u := range changed {
		for {
			log.Debugf("====>status change service: %+v", *u.Service)
			log.Debugf("============>fired cw.onChange<====")
			for _, f := range cw.onChange {
				f(u.Service.Address, u.Service.Port, EV_CHANGE)
			}
			break
		}
	}
}

func (cw *ConsulWatcher) dialDelete(addrs []*consul.ServiceEntry) {
	deleted := getDelete(cw.addrs, addrs)
	//如果发生改变的服务里面有leader，并且不是自己，则执行重新选leader
	for _, u := range deleted {
		for {
			log.Debugf("====>delete service: %+v", *u.Service)
			log.Debugf("============>fired cw.onChange<====")
			for _, f := range cw.onChange {
				f(u.Service.Address, u.Service.Port, EV_DELETE)
			}
			break
		}
	}
}

func (cw *ConsulWatcher) dialAdd(addrs []*consul.ServiceEntry) {
	added := getDelete(addrs, cw.addrs)
	//如果发生改变的服务里面有leader，并且不是自己，则执行重新选leader
	for _, u := range added {
		for {
			log.Debugf("====>delete service: %+v", *u.Service)
			log.Debugf("============>fired cw.onChange<====")
			for _, f := range cw.onChange {
				f(u.Service.Address, u.Service.Port, EV_ADD)
			}
			break
		}
	}
}

// queryConsul is helper function to query consul
func (cw *ConsulWatcher) queryConsul(q *consul.QueryOptions) ([]*consul.ServiceEntry, uint64, error) {
	// query consul
	cs, meta, err := cw.health.Service("wing-binlog-go-subscribe", "", false, q)
	if err != nil {
		return nil, 0, err
	}
	return cs, meta.LastIndex, nil
}

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

func (cw *ConsulWatcher) getMembers() ([]*consul.ServiceEntry, *consul.QueryMeta, error) {
	return cw.cc.Health().Service("wing-binlog-go-subscribe", "", true, nil)
}

func (cw *ConsulWatcher) getConnects(ip string, port int) uint64 {
	key := fmt.Sprintf("connects/%v/%v", ip, port)

	k, _, err := cw.cc.KV().Get(key, nil)
	if err != nil {
		log.Errorf("%v", err)
		return 0
	}
	return binary.LittleEndian.Uint64(k.Value)
}

