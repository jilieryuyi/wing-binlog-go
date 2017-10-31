package main

import (
	"fmt"
	"subscribe"
	"library/workers"
	"library/base"
	"library"
	"library/path"
	"runtime"
	"library/debug"
	_ "library/std"
	"database/sql"
	_ "mysql"
	//"log"
)

func main() {
	//std.Reset()
	cpu := runtime.NumCPU()
	debug.Print("cpu num: ", cpu)
	//指定cpu为多核运行
	runtime.GOMAXPROCS(cpu)

	current_path := path.GetCurrentPath()
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



	user     := "root"
	password := "123456"
	db, err  := sql.Open(
		"mysql",
		user+":"+ password+"@tcp(127.0.0.1:3306)/xsl?charset=utf8")

	if (nil != err) {
		panic(err);
		return;
	}

	defer db.Close();

	blog := library.Binlog{db}
	blog.Register(999)



	binlog.Start(notify);
	binlog.Loop(notify);
}
