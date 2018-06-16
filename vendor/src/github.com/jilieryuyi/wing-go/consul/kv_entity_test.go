package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
)
func TestNewKvEntity(t *testing.T) {
	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"

	client, _ := api.NewClient(config)
	e := NewKvEntity(client.KV(), "a", []byte("a"))
	_, err := e.Set()

	if err != nil {
		t.Errorf("set error")
	}

	_, err = e.Get()

	if err != nil {
		t.Errorf("get error")
	}

	_, err = e.Delete()

	if err != nil {
		t.Errorf("delete error")
	}

	_, err = e.Get()

	if err == nil {
		t.Errorf("get error")
	}
}
