package ip

import (
    "testing"
)

func TestLocal(t *testing.T) {
    ip , err := Local()
    if ip == "" || err != nil {
        t.Error("获取ip错误")
    } else {
        t.Log("测试通过，ip: " + ip)
    }
}
