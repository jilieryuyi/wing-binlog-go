package cluster

import (
	consulkv "github.com/armon/consul-kv"
	log "github.com/sirupsen/logrus"
	http "net/http"
	"time"
	"sync"
	"os"
	"strings"
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
	PREFIX_KEEPALIVE = "wing/binlog/keepalive/"
	PREFIX_NODE = "wing/binlog/node/"
	STATUS_ONLINE = "online"
	STATUS_OFFLINE = "offline"
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
		_, v, err := con.client.Get(PREFIX_KEEPALIVE + con.key)
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
		con.writeMember()
	}
	return con
}

func (con *Consul) writeMember() {
	if !con.enable {
		return
	}
	con.lock.Lock()
	defer con.lock.Unlock()
	hostname, err := os.Hostname()
	if err != nil {
		hostname = con.key
	}
	h  := []byte(hostname)
	hl := len(h)
	k  := []byte(con.key)
	kl := len(k)
	d  := make([]byte, 1 + 2 + hl + kl + 2)
	i  := 0
	d[i] = byte(con.isLock);i++
	d[i] = byte(hl);i++
	d[i] = byte(hl >> 8);i++
	for _, b := range h {
		d[i] = b; i++
	}
	d[i] = byte(kl); i++
	d[i] = byte(kl >> 8); i++
	for _, b := range k {
		d[i] = b; i++
	}
	con.client.Put(PREFIX_NODE + con.key, d, 0)
}

func (con *Consul) getMemberStatus(key string) string {
	_, v, err := con.client.Get(PREFIX_KEEPALIVE + key)
	if err == nil && v != nil {
		t := int64(v.Value[0]) | int64(v.Value[1])<<8 |
			int64(v.Value[2])<<16 | int64(v.Value[3])<<24 |
			int64(v.Value[4])<<32 | int64(v.Value[5])<<40 |
			int64(v.Value[6])<<48 | int64(v.Value[7])<<56
		if time.Now().Unix()-t > 3 {
			return STATUS_OFFLINE
		} else {
			return STATUS_ONLINE
		}
	}
	return STATUS_OFFLINE
}

func (con *Consul) ClearOfflineMembers() {
	//这里要先获取nodes
	//然后再获取keelive
	//如果keepalive中没有或者超时的都要清理掉
	members := con.GetMembers()
	log.Debugf("ClearOfflineMembers called")
	_, pairs, err := con.client.List(PREFIX_KEEPALIVE)
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
	_, members, err := con.client.List(PREFIX_NODE)
	if err != nil {
		return nil
	}
	m := make([]*ClusterMember, len(members))
	for i, member := range members {
		log.Debugf("key ==> %s", member.Key)
		m[i] = &ClusterMember{}
		m[i].IsLeader = int(member.Value[0]) == 1
		hl := int(member.Value[1]) | int(member.Value[2]) << 8
		m[i].Hostname = string(member.Value[3:2+hl])
		kl := int(member.Value[hl+3]) | int(member.Value[hl+4]) << 8
		m[i].Session = string(member.Value[hl+5:5+hl+kl])
		m[i].Status = con.getMemberStatus(m[i].Session)
	}
	return m
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
		con.client.Put(PREFIX_KEEPALIVE + con.key, r, 0)
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
			time.Sleep(time.Second * 3)
			continue
		}
		con.lock.Unlock()
		_, pairs, err := con.client.List(PREFIX_KEEPALIVE)
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

func (con *Consul) Close() {
	if !con.enable {
		return
	}
	con.Delete(PREFIX_KEEPALIVE + con.key)
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
