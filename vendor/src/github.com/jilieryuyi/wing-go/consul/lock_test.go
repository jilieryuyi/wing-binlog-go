package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
)

func TestNewLock(t *testing.T) {
	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"

	client, _ := api.NewClient(config)
	key       := "test"
	timeout   := int64(10)
	session   := NewSession(client.Session())
	sessionId, err := session.Create(10)
	if err != nil {
		t.Errorf("session create error")
	}
	lock      := NewLock(sessionId, client.KV())

	err = lock.Delete(key)
	if err != nil {
		t.Errorf("delete lock error")
	}

	success, err := lock.Lock(key, timeout)
	if err != nil || !success {
		t.Errorf("lock error")
	}

	success, err = lock.Unlock(key)
	if err != nil || !success {
		t.Errorf("unlock lock error")
	}

	err = lock.Delete(key)
	if err != nil {
		t.Errorf("delete lock error")
	}

}
