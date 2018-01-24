package cluster

import (
	//"encoding/json"
	//"errors"
	//"library/http"
	log "github.com/sirupsen/logrus"
	"github.com/hashicorp/consul/api"
)
//var IDErr = errors.New("ID key does not exists")
// create a session, use for lock a key
//func (con *Consul) createSession() (string, error) {
//	if !con.enable {
//		return "", nil
//	}
//	request := http.NewHttp("http://" + con.serviceIp + "/v1/session/create")
//	res, err := request.Put([]byte("{\"Name\": \"" + SESSION + "\"}"))
//	if err != nil {
//		return "", err
//	}
//	var arr interface{}
//	err = json.Unmarshal(res, &arr)
//	if err != nil {
//		return "", err
//	}
//	m := arr.(map[string] interface{});
//	id, ok := m["ID"]
//	if !ok {
//		return "", IDErr
//	}
//	log.Debugf("session id: %s", id.(string))
//	return id.(string), nil
//}

// lock if success, the current will be a leader
func (con *Consul) Lock() bool {
	if !con.enable {
		return true
	}
	if con.Session.ID == "" {
		con.Session.create()
	}
	if con.Session.ID == "" {
		log.Errorf("error: %v", ErrorSessionEmpty)
		return false
	}
	//key string, value []byte, sessionID string
	p := &api.KVPair{Key: LOCK, Value: nil, Session: con.Session.ID}
	success, _, err := con.Kv.Acquire(p, nil)
	if err != nil {
		log.Errorf("lock error: %+v", err)
	}
	if success {
		con.lock.Lock()
		con.isLock = 1
		con.lock.Unlock()
	}
	return success

	//defer con.writeMember()
	//lockApi := "http://" + con.serviceIp +"/v1/kv/" + LOCK + "?acquire=" + con.session
	//request := http.NewHttp(lockApi)
	//res, err := request.Put(nil)
	//if err != nil {
	//	log.Debugf("lock error: %+v", err)
	//	return false
	//}
	//log.Debugf("lock return: %s", string(res))
	////con.startLock<-struct{}{}
	//if string(res) == "true" {
	//	con.lock.Lock()
	//	con.isLock = 1
	//	con.lock.Unlock()
	//	return true
	//}
	//return false
}

// unlock
func (con *Consul) Unlock() bool {
	if !con.enable {
		return true
	}
	if con.Session.ID == "" {
		con.Session.create()
	}
	if con.Session.ID == "" {
		log.Errorf("error: %v", ErrorSessionEmpty)
		return false
	}
	p := &api.KVPair{Key: LOCK, Value: nil, Session: con.Session.ID}
	success, _, err := con.Kv.Release(p, nil)
	if err != nil {
		log.Errorf("lock error: %+v", err)
	}
	if success {
		con.lock.Lock()
		con.isLock = 0
		con.lock.Unlock()
	}
	return success//err == nil && success, err

	//defer con.writeMember()
	//unlockApi := "http://" + con.serviceIp +"/v1/kv/" + LOCK + "?release=" + con.session
	//request := http.NewHttp(unlockApi)
	//res, err := request.Put(nil)
	//if err != nil {
	//	log.Errorf("Unlock error: %+v", err)
	//	return false, err
	//}
	//log.Debugf("unlock return: %s", string(res))
	//if string(res) == "true" {
	//	con.lock.Lock()
	//	con.isLock = 0
	//	con.lock.Unlock()
	//	return true, nil
	//}
	//return false, nil
}

// delete a lock
func (con *Consul) Delete(key string) error {
	if !con.enable {
		return nil
	}
	if con.Session.ID == "" {
		con.Session.create()
	}
	if con.Session.ID == "" {
		return nil
	}
	_, err := con.Kv.Delete(key, nil)
	if err == nil {
		con.lock.Lock()
		if key == LOCK && con.isLock == 1 {
			con.isLock = 0
		}
		con.lock.Unlock()
	}
	return err

	//defer con.writeMember()
	//url := "http://" + con.serviceIp +"/v1/kv/" + key
	//request := http.NewHttp(url)
	//res, err := request.Delete()
	//if err != nil {
	//	log.Debugf("delete err: %+v", err)
	//	return
	//}
	//log.Debugf("delete %s return---%s", key, string(res))
	//con.lock.Lock()
	//if key == LOCK && con.isLock == 1 {
	//	con.isLock = 0
	//}
	//con.lock.Unlock()
}
