package main

import (
	"subscribe"
	"library/workers"
	"library/base"
	"library"
	"runtime"
	"library/debug"
	"database/sql"
	_ "mysql"
	"strconv"
)

func main() {

	//mac 输出 amd64 darwin
	debug.Print(runtime.GOARCH, runtime.GOOS)

	//library.Reset()
	cpu := runtime.NumCPU()
	debug.Print("cpu num: ", cpu)
	//指定cpu为多核运行
	runtime.GOMAXPROCS(cpu)

	current_path := library.GetCurrentPath()
	debug.Print(current_path)

	config_obj := &library.Ini{current_path+"/config/mysql.ini"}
	config     := config_obj.Parse();
	if config == nil {
		return
	}

	debug.Print(config)

	user     := string(config["mysql"]["user"].(string))
	password := string(config["mysql"]["password"].(string))
	port     := string(config["mysql"]["port"].(string))
	host     := string(config["mysql"]["host"].(string))
	slave_id_str := string(config["client"]["slave_id"].(string))
	slave_id, _:= strconv.Atoi(slave_id_str)
	db_name  := string(config["mysql"]["db_name"].(string))
	charset  := string(config["mysql"]["charset"].(string))
	db, err  := sql.Open("mysql", user+":"+ password+"@tcp(" + host + ":" + port + ")/" + db_name + "?charset=" + charset)

	if (nil != err) {
		debug.Print(err);
		return;
	}

	defer db.Close();

	blog := library.Binlog{db}
	blog.Register(slave_id)



	redis := &subscribe.Redis{}
	tcp   := &subscribe.Tcp{}

	//subscribes
	notify := []base.Subscribe{redis, tcp}
	binlog := &workers.Binlog{}

	defer binlog.End(notify);

	binlog.Start(notify);
	binlog.Loop(notify);
}
