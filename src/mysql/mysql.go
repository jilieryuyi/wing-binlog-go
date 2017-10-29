package main

import (
	"fmt"
	"subscribe"
	"library/workers"
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

	redis := &subscribe.Redis{}
	//redis.OnChange(data);

	tcp := &subscribe.Tcp{}
	//tcp.OnChange(data)

	//subscribes
	notify := []base.Subscribe{redis, tcp}

	binlog := &workers.Binlog{}
	binlog.Loop(notify);

}
