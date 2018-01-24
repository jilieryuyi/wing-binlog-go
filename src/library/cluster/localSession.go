package cluster

import (
	"library/path"
	"library/file"
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
	session := fmt.Sprintf("%d-%s-%s-%s", time.Now().Unix(),
		GetRandomString(4), GetRandomString(4), GetRandomString(4))
	dir := path.GetParent(sessionFile)
	path.Mkdir(dir)
	n := file.Write(sessionFile, session, false)
	if n != len(session) {
		return ""
	}
	return session
}
