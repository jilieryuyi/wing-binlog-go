package binlog

import (
	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)
// lock if success, the current will be a leader
func (h *Binlog) Lock() bool {
	if !h.enable {
		return true
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		log.Errorf("error: %v", ErrorSessionEmpty)
		return false
	}
	//key string, value []byte, sessionID string
	p := &api.KVPair{Key: LOCK, Value: nil, Session: h.Session.ID}
	success, _, err := h.Kv.Acquire(p, nil)
	if err != nil {
		log.Errorf("lock error: %+v", err)
	}
	if success {
		h.lock.Lock()
		h.isLock = 1
		h.lock.Unlock()
	}
	return success
}

// unlock
func (h *Binlog) Unlock() bool {
	if !h.enable {
		return true
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		log.Errorf("error: %v", ErrorSessionEmpty)
		return false
	}
	p := &api.KVPair{Key: LOCK, Value: nil, Session: h.Session.ID}
	success, _, err := h.Kv.Release(p, nil)
	if err != nil {
		log.Errorf("lock error: %+v", err)
	}
	if success {
		h.lock.Lock()
		h.isLock = 0
		h.lock.Unlock()
	}
	return success
}

// delete a lock
func (h *Binlog) Delete(key string) error {
	if !h.enable {
		return nil
	}
	if h.Session.ID == "" {
		h.Session.create()
	}
	if h.Session.ID == "" {
		return nil
	}
	_, err := h.Kv.Delete(key, nil)
	if err == nil {
		h.lock.Lock()
		if key == LOCK && h.isLock == 1 {
			h.isLock = 0
		}
		h.lock.Unlock()
	}
	return err
}
