package consul

import (
	"github.com/hashicorp/consul/api"
	"fmt"
)

type Session struct {
	session *api.Session
}
type ISession interface {
	Create(timeout int64) (string, error)
	Destroy(sessionId string) error
	Renew(sessionId string) error
}

func NewSession(session *api.Session) ISession {
	s := &Session{
		session:session,
	}
	return s
}

// create a session
// timeout unit is seconds
// return session id and error, if everything is ok, error should be nil
func (session *Session) Create(timeout int64) (string, error) {
	se := &api.SessionEntry{
		Behavior : api.SessionBehaviorDelete,
	}
	// timeout min value is 10 seconds
	if timeout > 0 && timeout < 10 {
		timeout = 10
	}
	if timeout > 0 {
		se.TTL = fmt.Sprintf("%ds", timeout)
	}
	ID, _, err := session.session.Create(se, nil)
	return ID, err
}

// destory a session
// sessionId is the value return from Create
func (session *Session) Destroy(sessionId string) error {
	_, err := session.session.Destroy(sessionId, nil)
	return err
}

// refresh a session
// sessionId is the value return from Create
func (session *Session) Renew(sessionId string) error {
	_, _, err := session.session.Renew(sessionId, nil)
	return err
}
