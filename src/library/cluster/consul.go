package cluster

import (
	consulkv "github.com/armon/consul-kv"
	log "github.com/sirupsen/logrus"
	http "net/http"
	"time"
	"sync"
)
type Consul struct {
	Cluster
	client *consulkv.Client
	serviceIp string
	session string
	isLock int
	lock *sync.Mutex
	onLeaderCallback func()
	onPosChange func([]byte)
	key string
	enable bool
	//startLock chan struct{}
}
const (
	POS_KEY = "wing/binlog/pos"
	LOCK = "wing/binlog/lock"
	SESSION = "wing/binlog/session"
)

func NewConsul(onLeaderCallback func(), onPosChange func([]byte)) *Consul{
	config, err := GetConfig()
	log.Debugf("cluster config: %+v", *config.Consul)
	if err != nil {
		log.Panicf("new consul client with error: %+v", err)
	}
	con := &Consul{
		serviceIp:config.Consul.ServiceIp,
		isLock:0,
		lock:new(sync.Mutex),
		key:GetSession(),
		//startLock:make(chan struct{}),
		onLeaderCallback:onLeaderCallback,
		onPosChange:onPosChange,
		enable:config.Enable,
	}

	if con.enable {
		con.session, err = con.createSession()
		if err != nil {
			log.Panicf("create consul session with error: %+v", err)
		}
		//http.DefaultClient.Timeout = time.Second * 6
		kvConfig := &consulkv.Config{
			Address:    config.Consul.ServiceIp,
			HTTPClient: http.DefaultClient,
		}
		con.client, err = consulkv.NewClient(kvConfig)
		if err != nil {
			log.Panicf("new consul client with error: %+v", err)
		}

		// check self is locked in start
		// if is locked, try unlock
		_, v, err := con.client.Get("wing/binlog/keepalive/" + con.key)
		if err == nil && v != nil {
			t := int64(v.Value[0]) | int64(v.Value[1])<<8 |
				int64(v.Value[2])<<16 | int64(v.Value[3])<<24 |
				int64(v.Value[4])<<32 | int64(v.Value[5])<<40 |
				int64(v.Value[6])<<48 | int64(v.Value[7])<<56
			isLock := 0
			if len(v.Value) > 8 {
				isLock = int(v.Value[8])
			}
			if time.Now().Unix()-t > 3 && isLock == 1 {
				con.Unlock()
				con.Delete(LOCK)
				con.Delete(v.Key)
			}
		}

		//超时检测，即检测leader是否挂了，如果挂了，要重新选一个leader
		//如果当前不是leader，重新选leader。leader不需要check
		//如果被选为leader，则还需要执行一个onLeader回调
		go con.checkAlive()
		//还需要一个keepalive
		go con.keepalive()
		//还需要一个检测pos变化回调，即如果不是leader，要及时更新来自leader的pos变化
		go con.watch()
	}
	return con
}

func (con *Consul) keepalive() {
	if !con.enable {
		return
	}
	r := make([]byte, 9)
	for {
		t := time.Now().Unix()
		r[0] = byte(t)
		r[1] = byte(t >> 8)
		r[2] = byte(t >> 16)
		r[3] = byte(t >> 24)
		r[4] = byte(t >> 32)
		r[5] = byte(t >> 40)
		r[6] = byte(t >> 48)
		r[7] = byte(t >> 56)
		con.lock.Lock()
		r[8] = byte(con.isLock)
		con.lock.Unlock()
		con.client.Put("wing/binlog/keepalive/" + con.key, r, 0)
		//log.Debugf("write keepalive %d", t)
		time.Sleep(time.Second * 1)
	}
}

func (con *Consul) checkAlive() {
	if !con.enable {
		return
	}
	for {
		con.lock.Lock()
		if con.isLock == 1 {
			con.lock.Unlock()
			// leader does not need check
			//log.Debugf("checkAlive is leader")
			time.Sleep(time.Second * 3)
			continue
		}
		con.lock.Unlock()
		_, pairs, err := con.client.List("wing/binlog/keepalive")
		if err != nil {
			log.Errorf("checkAlive with error：%#v", err)
			time.Sleep(time.Second)
			continue
		}
		if pairs == nil {
			time.Sleep(time.Second * 3)
			continue
		}

		reLeader := true
		leaderCount := 0
		for _, v := range pairs {
			if v.Value == nil {
				log.Debugf("%+v", v)
				log.Debug("checkAlive value nil")
				continue
			}
			t := int64(v.Value[0]) | int64(v.Value[1]) << 8 |
					int64(v.Value[2]) << 16 | int64(v.Value[3]) << 24 |
					int64(v.Value[4]) << 32 | int64(v.Value[5]) << 40 |
					int64(v.Value[6]) << 48 | int64(v.Value[7]) << 56
			isLock := 0
			if len(v.Value) > 8 {
				isLock = int(v.Value[8])
			}
			if isLock == 1 {
				reLeader = false
				leaderCount++
			}
			if time.Now().Unix() - t > 3 {
				con.Delete(v.Key)
				if isLock == 1 {
					reLeader = true
				}
			}
		}
		if reLeader || leaderCount > 1 {
			log.Warnf("leader maybe leave, try to create a new leader")
			//con.Unlock()
			con.Delete(LOCK)
			if con.Lock() {
				if con.onLeaderCallback != nil {
					con.onLeaderCallback()
				}
			}
		}
		time.Sleep(time.Second * 3)
	}
}

func (con *Consul) watch() {
	if !con.enable {
		return
	}
	//select {
	//	case <-con.startLock:
	//}
	//log.Debugf("watch start...")
	for {
		con.lock.Lock()
		//log.Debugf("watch %d", con.isLock)
		if con.isLock == 1 {
			con.lock.Unlock()
			// leader does not need watch
			time.Sleep(time.Second*3)
			continue
		}
		con.lock.Unlock()
		meta, _, err := con.client.List("wing/binlog/pos")
		if err != nil {
			log.Errorf("watch chang with error：%#v", err)
			time.Sleep(time.Second)
			continue
		}
		if meta == nil {
			time.Sleep(time.Second)
			continue
		}
		//if d != nil {
		//	log.Debugf("%+v", d)
		//	for _, vv := range d {
		//		if vv == nil {
		//			continue
		//		}
		//		log.Debugf("===>%+v", vv)
		//		con.onPosChange(vv.Value)
		//	}
		//}
		_, v, err := con.client.WatchGet("wing/binlog/pos", meta.ModifyIndex)
		if err != nil {
			log.Errorf("watch chang with error：%#v, %+v", err, v)
			time.Sleep(time.Second)
			continue
		}
		if v == nil {
			time.Sleep(time.Second)
			continue
		}
		if v.Value == nil {
			continue
		}
		con.onPosChange(v.Value)
		time.Sleep(time.Microsecond * 1)
	}
}

//func (con *Consul) RegisterOnLeaderCallback(fun func()) {
//	con.onLeaderCallback = fun
//}
//
//func (con *Consul) RegisterOnPosChangeCallback(fun func([]byte)) {
//	con.onPosChange = fun
//}

func (con *Consul) Close() {
	if !con.enable {
		return
	}
	con.Delete("wing/binlog/keepalive/" + con.key)
	log.Debugf("current is leader %d", con.isLock)
	con.lock.Lock()
	l := con.isLock
	con.lock.Unlock()
	if l == 1 {
		log.Debugf("delete lock %s", LOCK)
		con.Unlock()
		con.Delete(LOCK)
	}
}

func (con *Consul) Write(data []byte) bool {
	if !con.enable {
		return true
	}
	log.Debugf("write consul kv: %s, %v", POS_KEY, data)
	err := con.client.Put(POS_KEY, data, 0)
	if err != nil {
		log.Errorf("write consul kv with error: %+v", err)
	}
	return nil == err
}

func (con *Consul) Read() []byte {
	if !con.enable {
		return nil
	}
	_ ,v, err := con.client.Get(POS_KEY)
	if err != nil {
		log.Errorf("write consul kv with error: %+v", err)
		return nil
	}
	return v.Value
}
