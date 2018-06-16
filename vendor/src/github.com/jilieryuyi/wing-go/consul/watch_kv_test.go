package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
	"fmt"
	"bytes"
	"time"
)
func TestNewWatchKv(t *testing.T) {
	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"
	value := []byte("hello word")
	client, _ := api.NewClient(config)
	watch := NewWatchKv(client.KV(), "test")
	watch.Watch(func(bt []byte) {
		fmt.Println("new data: ", string(bt))
		if !bytes.Equal(value, bt) {
			t.Errorf("watch error")
		}
	})

	kv := NewKvEntity(client.KV(), "test/a", value)
	kv.Set()

	a := time.After(time.Second * 3)
	select {
	case <- a:
		kv.Delete()
	}
}
