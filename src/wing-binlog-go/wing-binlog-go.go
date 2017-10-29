package main

import (
	"fmt"
	"subscribe"
	"library/workers"
	"library/base"
	"library"
	"library/path"
)

func main() {

	current_path := path.GetCurrentPath();
	fmt.Println(current_path)

	config_obj := &library.Config{current_path+"/app.json"}
	config := config_obj.Parse();
	fmt.Println(config)

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

	defer func() {
		//结束时清理资源
		binlog.End(notify);
	}()

	binlog.Start(notify);
	binlog.Loop(notify);
}
