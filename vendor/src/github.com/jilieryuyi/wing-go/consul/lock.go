package consul

import (
	"github.com/hashicorp/consul/api"
)

type Lock struct {
	sessionId string
	kv *api.KV
}

type ILock interface {
	Lock(key string, timeout int64) (bool, error)
	Unlock(key string) (bool, error)
	Delete(key string) error
}

func NewLock(sessionId string, kv *api.KV) ILock {
	con := &Lock{
		sessionId: sessionId,
		kv: kv,
	}
	return con
}

// timeout seconds, max lock time, min value is 10 seconds
func (con *Lock) Lock(key string, timeout int64) (bool, error) {
	p := &api.KVPair{Key: key, Value: nil, Session: con.sessionId}
	success, _, err := con.kv.Acquire(p, nil)
	return success, err
}

// unlock
func (con *Lock) Unlock(key string) (bool, error) {
	p := &api.KVPair{Key: key, Value: nil, Session: con.sessionId}
	success, _, err := con.kv.Release(p, nil)
	return success, err

}

// force unlock
func (con *Lock) Delete(key string) error {
	_, err := con.kv.Delete(key, nil)
	return err
}