package consul

import (
	"github.com/hashicorp/consul/api"
)

type SessionEntity struct {
	session ISession
	timeout int64
	ID string
}

func NewSessionEntity(session *api.Session, timeout int64) *SessionEntity {
	if timeout > 0 && timeout < 10 {
		timeout = 10
	}
	s := &SessionEntity{
		session: NewSession(session),
		timeout:timeout,
	}
	return s
}

// create a session
// timeout unit is seconds
// return session id and error, if everything is ok, error should be nil
func (session *SessionEntity) Create() (string, error) {
	var err error
	session.ID, err = session.session.Create(session.timeout)
	return session.ID, err
}

// destory a session
// sessionId is the value return from Create
func (session *SessionEntity) Destroy() error {
	return session.session.Destroy(session.ID)
}

// refresh a session
// sessionId is the value return from Create
func (session *SessionEntity) Renew() error {
	return session.session.Renew(session.ID)
}
