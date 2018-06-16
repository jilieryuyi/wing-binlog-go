package consul

import (
	"testing"
)

func TestLeader_Register(t *testing.T) {
	serviceName := "service-test"
	lockKey := "test"
	address := "127.0.0.1:8500"


	leader1 := NewLeader(
		address,
		lockKey,
		serviceName,
		"127.0.0.1",
		7770,
	)
	_, err := leader1.Register()
	if err != nil {
		t.Errorf("register err")
	}

	err = leader1.UpdateTtl()
	if err != nil {
		t.Errorf("UpdateTtl err")
	}

	err = leader1.Deregister()
	if err != nil {
		t.Errorf("Deregister err")
	}

	err = leader1.UpdateTtl()
	if err == nil {
		t.Errorf("UpdateTtl err")
	}
}