package ip

import (
	"testing"
	"fmt"
)

func TestLocal(t *testing.T) {
	ip, err := Local()
	fmt.Printf("local ip===========%s==========\n", ip)
	if ip == "" || err != nil {
		t.Error("获取ip错误")
	} else {
		t.Log("测试通过，ip: " + ip)
	}
}
