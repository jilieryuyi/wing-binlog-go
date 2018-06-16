package consul

import (
	"github.com/hashicorp/consul/api"
)

type LockEntity struct {
	sessionId string
	kv *api.KV
	key string
	timeout int64
	lock ILock
}

func NewLockEntity(sessionId string, kv *api.KV, key string, timeout int64) *LockEntity {
	lock := NewLock(sessionId, kv)
	return &LockEntity{
		lock:lock,
		sessionId:sessionId,
		kv:kv,
		key:key, timeout:timeout,
	}
}

// timeout seconds, max lock time, min value is 10 seconds
func (con *LockEntity) Lock() (bool, error) {
	return con.lock.Lock(con.key, con.timeout)
}

// unlock
func (con *LockEntity) Unlock() (bool, error) {
	return con.lock.Unlock(con.key)

}

// force unlock
func (con *LockEntity) Delete() error {
	return con.lock.Delete(con.key)
}
