package cluster

import (
	"github.com/hashicorp/consul/api"
)

type Session struct {
	Address string
	ID string
	s *api.Session
}

// timeout 单位为秒
func (ses *Session) create() {
	se := &api.SessionEntry{
		Behavior : api.SessionBehaviorDelete,
		TTL: "60s",
	}
	ID, _, err := ses.s.Create(se, nil)
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
		return ErrorSessionEmpty
	}
	_, _, err = ses.s.Renew(ses.ID, nil)
	return err
}

func (ses *Session) delete() (err error) {
	_, err = ses.s.Destroy(ses.ID, nil)
	return err
}
