package binlog

import (
	"github.com/hashicorp/consul/api"
)

type Session struct {
	Address string
	ID string
	handler *api.Session
}

// timeout 单位为秒
func (ses *Session) create() {
	se := &api.SessionEntry{
		Behavior : api.SessionBehaviorDelete,
		TTL: "86400s",
	}
	ID, _, err := ses.handler.Create(se, nil)
	if err != nil {
		return
	}
	ses.ID = ID
}

func (ses *Session) renew() (err error) {
	if ses.ID == "" {
		ses.create()
	}
	if ses.ID == "" {
		return sessionEmpty
	}
	_, _, err = ses.handler.Renew(ses.ID, nil)
	return err
}

func (ses *Session) delete() (err error) {
	_, err = ses.handler.Destroy(ses.ID, nil)
	return err
}
