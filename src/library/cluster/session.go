package cluster

import (
	"library/path"
	"library/file"
	"fmt"
	"time"
	"math/rand"
)

func GetSession() string {
	sessionFile := path.CurrentPath + "/cache/session"
	if file.Exists(sessionFile) {
		data := file.Read(sessionFile)
		if data != "" {
			return data
		}
	}
	//write a new session
	session := fmt.Sprintf("%d-%d-%d-%d", time.Now().Unix(), rand.Intn(9999), rand.Intn(9999), rand.Intn(9999))
	dir := path.GetParent(sessionFile)
	path.Mkdir(dir)
	n := file.Write(sessionFile, session, false)
	if n != len(session) {
		return ""
	}
	return session
}
