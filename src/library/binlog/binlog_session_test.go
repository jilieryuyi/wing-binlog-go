package binlog

import (
	"testing"
	"fmt"
	"library/app"
)

func TestGetSession(t *testing.T) {
	session := app.GetKey(app.CachePath + "/session")//GetSession()
	fmt.Println("session=", session)
	if session == "" {
		t.Error("get session error")
	}
	session2 :=  app.GetKey(app.CachePath + "/session")
	fmt.Println("session=", session2)
	if session != session2 {
		t.Error("get session error")
	}
}
