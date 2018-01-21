package cluster

import (
	"library/path"
	"library/file"
	"fmt"
	"time"
	"math/rand"
	log "github.com/sirupsen/logrus"
)

func GetRandomString(l int) string{
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	bl := len(bytes)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(bl)])
	}
	return string(result)
}

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
