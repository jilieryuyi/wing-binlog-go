package cluster

import (
	"library/path"
	"library/file"
	"library/util"
	"fmt"
	"time"
	log "github.com/sirupsen/logrus"
)

func GetSession() string {
	sessionFile := path.CurrentPath + "/cache/session"
	log.Debugf("session file: %s", sessionFile)
	if file.Exists(sessionFile) {
		data := file.Read(sessionFile)
		if data != "" {
			return data
		}
	}
	//write a new session
	session := fmt.Sprintf("%d-%s", time.Now().Unix(), util.RandString(64))
	dir := path.GetParent(sessionFile)
	path.Mkdir(dir)
	n := file.Write(sessionFile, session, false)
	if n != len(session) {
		return ""
	}
	return session
}
