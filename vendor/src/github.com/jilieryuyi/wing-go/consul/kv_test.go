package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
	"bytes"
)

func TestNewKv(t *testing.T) {
	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"

	client, _ := api.NewClient(config)
	kv := NewKv(client.KV())

	key   := "a"
	value := []byte("a")

	err := kv.Set(key, value)
	if err != nil {
		t.Errorf("set kv error")
	}

	v, err := kv.Get(key)
	if err != nil {
		t.Errorf("get kv error")
	}

	if !bytes.Equal(v, value) {
		t.Errorf("set kv error")
	}

	err = kv.Delete(key)
	if err != nil {
		t.Errorf("delete kv error")
	}

	v, err = kv.Get(key)
	if err == nil {
		t.Errorf("get kv error")
	}

	if bytes.Equal(v, value) {
		t.Errorf("get kv error")
	}
}
