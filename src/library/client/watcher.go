package client

import (
	"time"
	consul "github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"fmt"
	"encoding/binary"
)

//监听服务变化

// ConsulWatcher is the implementation of grpc.naming.Watcher
type ConsulWatcher struct {
	// cc: Consul Client
	cc *consul.Client
	// LastIndex to watch consul
	li uint64
	addrs []*consul.ServiceEntry//[]string
	health *consul.Health
	onChange []onChangeFunc
}

const (
	EV_ADD = 1
	EV_DELETE = 2 // need to delete service
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
	for {
		// watch consul
		addrs, li, err := cw.queryConsul(&consul.QueryOptions{WaitIndex: cw.li})
		if err != nil {
			log.Errorf("============>cw.queryConsul error: %+v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		if addrs == nil {
			log.Warnf("watch consul services return nil")
			addrs = make([]*consul.ServiceEntry, 0)
		}
		cw.dialDelete(addrs)
		cw.dialAdd(addrs)
		cw.addrs = addrs
		cw.li = li.LastIndex
	}
}

func (cw *ConsulWatcher) dialDelete(addrs []*consul.ServiceEntry) {
	deleted := getDelete(cw.addrs, addrs)
	for _, u := range deleted {
		for {
			//log.Debugf("====>delete service: %+v", *u.Service)
			log.Debugf("============>fired EV_DELETE cw.onChange<====")
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
			//log.Debugf("====>delete service: %+v", *u.Service)
			log.Debugf("============>fired EV_ADD cw.onChange<====")
			for _, f := range cw.onChange {
				f(u.Service.Address, u.Service.Port, EV_ADD)
			}
			break
		}
	}
}

// queryConsul is helper function to query consul
func (cw *ConsulWatcher) queryConsul(q *consul.QueryOptions) ([]*consul.ServiceEntry, *consul.QueryMeta, error) {
	// query consul
	cs, meta, err := cw.health.Service("wing-binlog-go-subscribe", "", true, q)
	if err != nil {
		return nil, nil, err
	}
	return cs, meta, nil
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

func (cw *ConsulWatcher) getMembers() ([]*consul.ServiceEntry, error) {
	c, i, e := cw.queryConsul(nil)
	if e == nil {
		cw.addrs = c
		cw.li = i.LastIndex
	}
	return c, e
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

