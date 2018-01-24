package cluster

import (
	log "github.com/sirupsen/logrus"
	"time"
	"sync"
	"os"
	"strings"
	"github.com/hashicorp/consul/api"
)
type Consul struct {
	Cluster
	serviceIp string
	Session *Session
	isLock int
	lock *sync.Mutex
	onLeaderCallback func()
	onPosChange func([]byte)
	sessionId string
	enable bool
	Client *api.Client
	Kv *api.KV
}
const (
	POS_KEY = "wing/binlog/pos"
	LOCK = "wing/binlog/lock"
	SESSION = "wing/binlog/session"
	PREFIX_KEEPALIVE = "wing/binlog/keepalive/"
	PREFIX_NODE = "wing/binlog/node/"
	STATUS_ONLINE = "online"
	STATUS_OFFLINE = "offline"
)

func NewConsul(onLeaderCallback func(), onPosChange func([]byte)) *Consul{
	log.Debugf("start cluster...")
	config, err := GetConfig()
	log.Debugf("cluster config: %+v", *config.Consul)
	if err != nil {
		log.Panicf("new consul client with error: %+v", err)
	}
	con := &Consul{
		serviceIp:config.Consul.ServiceIp,
		isLock:0,
		lock:new(sync.Mutex),
		sessionId:GetSession(),
		onLeaderCallback:onLeaderCallback,
		onPosChange:onPosChange,
		enable:config.Enable,
	}
	if con.enable {
		ConsulConfig := api.DefaultConfig()
		ConsulConfig.Address = config.Consul.ServiceIp
		var err error
		con.Client, err = api.NewClient(ConsulConfig)
		if err != nil {
			log.Panicf("create consul session with error: %+v", err)
		}
		con.Session = &Session {
			Address:config.Consul.ServiceIp,
			Client:con.Client,
		}
		con.Session.create()
		con.Kv = con.Client.KV()
		// check self is locked in start
		// if is locked, try unlock
		v, _, err := con.Kv.Get(PREFIX_KEEPALIVE + con.sessionId, nil)
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
		////还需要一个keepalive
		go con.keepalive()
		////还需要一个检测pos变化回调，即如果不是leader，要及时更新来自leader的pos变化
		go con.watch()
	}
	return con
}

func (con *Consul) keepalive() {
	if !con.enable {
		return
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = con.sessionId
	}
	h  := []byte(hostname)
	hl := len(h)
	k  := []byte(con.sessionId)
	kl := len(k)
	//d  := make([]byte, 1 + 2 + hl + kl + 2)
	r := make([]byte, 9 + 2 + hl + kl + 2)
	for {
		t := time.Now().Unix()
		i := 0
		//0-7 bytes
		r[i] = byte(t); i++
		r[i] = byte(t >> 8); i++
		r[i] = byte(t >> 16); i++
		r[i] = byte(t >> 24); i++
		r[i] = byte(t >> 32); i++
		r[i] = byte(t >> 40); i++
		r[i] = byte(t >> 48); i++
		r[i] = byte(t >> 56); i++
		con.lock.Lock()
		// 1 byte, index 8
		r[i] = byte(con.isLock); i++
		con.lock.Unlock()

		//2 bytes, index 9 -10
		r[i] = byte(hl);i++
		r[i] = byte(hl >> 8);i++
		for _, b := range h {
			r[i] = b; i++
		}
		r[i] = byte(kl); i++
		r[i] = byte(kl >> 8); i++
		for _, b := range k {
			r[i] = b; i++
		}
		con.Kv.Put(&api.KVPair{Key: PREFIX_KEEPALIVE + con.sessionId, Value: r}, nil)//client.Put(PREFIX_KEEPALIVE + con.key, r, 0)
		time.Sleep(time.Second * 1)
	}
}

func (con *Consul) ClearOfflineMembers() {
	//这里要先获取nodes
	//然后再获取keelive
	//如果keepalive中没有或者超时的都要清理掉
	members := con.GetMembers()
	log.Debugf("ClearOfflineMembers called")
	pairs, _, err := con.Kv.List(PREFIX_KEEPALIVE, nil)
	if err != nil || pairs == nil {
		log.Debugf("ClearOfflineMembers %+v,,%+v", pairs, err)
		return
	}
	if members != nil {
		for _, member := range members {
			find := false
			for _, v := range pairs {
				//key := "wing/binlog/keepalive/1516541226-7943-9879-4574"
				i := strings.LastIndex(v.Key, "/")
				if i > -1 {
					if v.Key[i+1:] == member.Session {
						//find
						find = true
					}
				}
			}
			if !find {
				log.Debugf("delete not in %s", member.Session)
				con.Delete(PREFIX_NODE + member.Session)
			}
		}
	}
	for _, v := range pairs {
		log.Debugf("key ==> %s", v.Key)
		t := int64(v.Value[0]) | int64(v.Value[1])<<8 |
			int64(v.Value[2])<<16 | int64(v.Value[3])<<24 |
			int64(v.Value[4])<<32 | int64(v.Value[5])<<40 |
			int64(v.Value[6])<<48 | int64(v.Value[7])<<56
		if time.Now().Unix()-t > 3 {
			log.Debugf("delete key %s", v.Key)
			con.Delete(v.Key)
			i := strings.LastIndex(v.Key, "/")
			if i > -1 {
				log.Debugf("delete key %s", PREFIX_NODE + v.Key[i+1:])
				con.Delete(PREFIX_NODE + v.Key[i+1:])
			}
		}
	}
}

func (con *Consul) GetMembers() []*ClusterMember {
	members,_, err := con.Kv.List(PREFIX_KEEPALIVE, nil)
	if err != nil {
		return nil
	}
	_hostname, err := os.Hostname()
	log.Debugf("hostname: ", _hostname)
	m := make([]*ClusterMember, len(members))
	for i, v := range members {
		log.Debugf("key ==> %s", v.Key)
		m[i] = &ClusterMember{}
		t := int64(v.Value[0]) | int64(v.Value[1])<<8 |
			int64(v.Value[2])<<16 | int64(v.Value[3])<<24 |
			int64(v.Value[4])<<32 | int64(v.Value[5])<<40 |
			int64(v.Value[6])<<48 | int64(v.Value[7])<<56
		if time.Now().Unix()-t > 3 {
			m[i].Status = STATUS_OFFLINE
		} else {
			m[i].Status = STATUS_ONLINE
		}
		m[i].IsLeader = int(v.Value[8]) == 1
		//9 - 10 is hostname len
		hl := int(v.Value[9]) | int(v.Value[10])<<8
		//hl := int(member.Value[1]) | int(member.Value[2]) << 8
		m[i].Hostname = string(v.Value[11:12+hl])
		//kl := int(v.Value[hl+11]) | int(v.Value[hl+12]) << 8
		m[i].Session = string(v.Value[hl+13:])
	}
	return m
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
			time.Sleep(time.Second * 3)
			continue
		}
		con.lock.Unlock()
		pairs, _, err := con.Kv.List(PREFIX_KEEPALIVE, nil)
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
	for {
		con.lock.Lock()
		if con.isLock == 1 {
			con.lock.Unlock()
			// leader does not need watch
			time.Sleep(time.Second * 3)
			continue
		}
		con.lock.Unlock()
		_, meta, err := con.Kv.Get(POS_KEY, nil)
		if err != nil {
			log.Errorf("watch pos change with error：%#v", err)
			time.Sleep(time.Second)
			continue
		}
		if meta == nil {
			time.Sleep(time.Second)
			continue
		}
		v, _, err := con.Kv.Get(POS_KEY, &api.QueryOptions{
			WaitIndex : meta.LastIndex,
			WaitTime : time.Second * 3,
		})
		if err != nil {
			log.Errorf("watch chang with error：%#v, %+v", err, v)
			time.Sleep(time.Second)
			continue
		}
		if v == nil {
			time.Sleep(time.Second)
			continue
		}
		con.onPosChange(v.Value)
		time.Sleep(time.Millisecond * 10)
	}
}

func (con *Consul) Close() {
	if !con.enable {
		return
	}
	con.Delete(PREFIX_KEEPALIVE + con.sessionId)
	log.Debugf("current is leader %d", con.isLock)
	con.lock.Lock()
	l := con.isLock
	con.lock.Unlock()
	if l == 1 {
		log.Debugf("delete lock %s", LOCK)
		con.Unlock()
		con.Delete(LOCK)
	}
	con.Session.delete()
}

func (con *Consul) Write(data []byte) bool {
	if !con.enable {
		return true
	}
	log.Debugf("write consul kv: %s, %v", POS_KEY, data)
	_, err := con.Kv.Put(&api.KVPair{Key: POS_KEY, Value: data}, nil)
	if err != nil {
		log.Errorf("write consul kv with error: %+v", err)
	}
	return nil == err
}

func (con *Consul) Read() []byte {
	if !con.enable {
		return nil
	}
	v ,_, err := con.Kv.Get(POS_KEY, nil)
	if err != nil {
		log.Errorf("write consul kv with error: %+v", err)
		return nil
	}
	return v.Value
}
