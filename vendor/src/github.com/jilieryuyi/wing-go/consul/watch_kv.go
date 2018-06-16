package consul

import (
	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
	"time"
)


type WatchKv struct {
	kv *api.KV
	prefix string
	notify []Notify
}

type Notify func(kv *api.KV, key string, data interface{})
type WatchKvOption func(k *WatchKv)
type IWatchKv interface {
	Watch(watch func([]byte))
}

func NewWatchKv(kv *api.KV, prefix string) IWatchKv {
	k := &WatchKv{
		prefix:prefix,
		kv:kv,
		notify:make([]Notify, 0),
	}
	return k
}

func (m *WatchKv) Watch(watch func([]byte)) {
	go func() {
		lastIndex := uint64(0)
		for {
			_, me, err := m.kv.List(m.prefix, nil)
			if err != nil || me == nil {
				log.Errorf("%+v", err)
				time.Sleep(time.Second)
				continue
			}
			lastIndex = me.LastIndex
			break
		}
		for {
			qp := &api.QueryOptions{WaitIndex: lastIndex}
			kp, me, e :=  m.kv.List(m.prefix, qp)
			if e != nil {
				log.Errorf("%+v", e)
				time.Sleep(time.Second)
				continue
			}
			lastIndex = me.LastIndex
			for _, v := range kp {
				if len(v.Value) == 0 {
					continue
				}
				watch(v.Value)
			}
		}
	}()
}
