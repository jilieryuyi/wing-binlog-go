package cluster

import (
	"encoding/json"
	"errors"
	"library/http"
	log "github.com/sirupsen/logrus"
)
var IDErr = errors.New("ID key does not exists")
// create a session, use for lock a key
func (con *Consul) createSession() (string, error) {
	request := http.NewHttp("http://" + con.serviceIp + "/v1/session/create")
	res, err := request.Put([]byte("{\"Name\": \"" + SESSION + "\"}"))
	if err != nil {
		return "", err
	}
	var arr interface{}
	err = json.Unmarshal(res, &arr)
	if err != nil {
		return "", err
	}
	m := arr.(map[string] interface{});
	id, ok := m["ID"]
	if !ok {
		return "", IDErr
	}
	log.Debugf("session id: %s", id.(string))
	return id.(string), nil
}

// lock if success, the current will be a leader
func (con *Consul) Lock() bool {
	lockApi := "http://" + con.serviceIp +"/v1/kv/" + LOCK + "?acquire=" + con.session
	request := http.NewHttp(lockApi)
	res, err := request.Put(nil)
	if err != nil {
		log.Debugf("lock error: %+v", err)
		return false
	}
	log.Debugf("lock return: %s", string(res))
	//con.startLock<-struct{}{}
	if string(res) == "true" {
		con.isLock = 1
		return true
	}
	return false
}

// unlock
func (con *Consul) Unlock() (bool, error) {
	unlockApi := "http://" + con.serviceIp +"/v1/kv/" + LOCK + "?release=" + con.session
	request := http.NewHttp(unlockApi)
	res, err := request.Put(nil)
	if err != nil {
		log.Errorf("Unlock error: %+v", err)
		return false, err
	}
	log.Debugf("unlock return: %s", string(res))
	if string(res) == "true" {
		con.isLock = 0
		return true, nil
	}
	return false, nil
}

// delete a lock
func (con *Consul) Delete(key string) {
	url := "http://" + con.serviceIp +"/v1/kv/" + key
	request := http.NewHttp(url)
	res, err := request.Delete()
	if err != nil {
		log.Debugf("delete err: %+v", err)
		return
	}
	log.Debugf("delete %s return---%s", key, string(res))
	con.lock.Lock()
	if key == LOCK && con.isLock == 1 {
		con.isLock = 0
	}
	con.lock.Unlock()
}
