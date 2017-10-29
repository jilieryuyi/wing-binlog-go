package main

import (
	"fmt"
	"subscribe/redis"
	"subscribe/tcp"
	"library/workers/binlog"
	"library/base"
)

type R struct{

}
func (r *R) Test() {
	fmt.Println("hello")
}
func main() {
	data := make(map[string] interface{})
	data["hello"] = "yuyi"
	fmt.Println(data)

	redis := &redis.Redis{}
	//redis.OnChange(data);

	tcp := &tcp.Tcp{}
	//tcp.OnChange(data)

	//subscribes
	notify := []base.Subscribe{redis, tcp}

	binlog := &binlog.Binlog{}
	binlog.Loop(notify);

}
