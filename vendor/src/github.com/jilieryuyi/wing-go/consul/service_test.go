package consul

import (
	"testing"
	"github.com/hashicorp/consul/api"
)

func TestNewService(t *testing.T) {
	address := "127.0.0.1:8500"
	config := api.DefaultConfig()
	config.Address = address
	client, _ := api.NewClient(config)
	agent := client.Agent()

	sev1 := NewService(agent, "test", "127.0.0.1", 7000)
	err := sev1.Register()
	if err != nil {
		t.Errorf("register error")
	}
	err = sev1.Deregister()
	if err != nil {
		t.Errorf("Deregister error")
	}
	sev1 = NewService(agent, "test", "127.0.0.1", 7001)
	err = sev1.Register()
	if err != nil {
		t.Errorf("register error")
	}
	err = sev1.Deregister()
	if err != nil {
		t.Errorf("Deregister error")
	}
	sev1 = NewService(agent, "test", "127.0.0.1", 7002, )
	err = sev1.Register()
	if err != nil {
		t.Errorf("register error")
	}
	err = sev1.Deregister()
	if err != nil {
		t.Errorf("Deregister error")
	}
}
