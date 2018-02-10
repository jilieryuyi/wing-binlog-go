package binlog

import (
	"github.com/hashicorp/consul/api"
	"library/path"
	"library/file"
	"fmt"
	"time"
	log "github.com/sirupsen/logrus"
	wstring "library/string"
	"library/app"
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

func GetSession() string {
	sessionFile := app.CachePath + "/session"
	log.Debugf("session file: %s", sessionFile)
	if file.Exists(sessionFile) {
		data := file.Read(sessionFile)
		if data != "" {
			return data
		}
	}
	//write a new session
	session := fmt.Sprintf("%d-%s", time.Now().Unix(), wstring.RandString(64))
	dir := path.GetParent(sessionFile)
	path.Mkdir(dir)
	n := file.Write(sessionFile, session, false)
	if n != len(session) {
		return ""
	}
	return session
}
