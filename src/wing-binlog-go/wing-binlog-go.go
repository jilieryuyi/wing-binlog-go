package main

import (
	"subscribe"
	"library/workers"
	"library/base"
	"library"
	"runtime"
	//"library/debug"
	"database/sql"
	_ "mysql"
	"strconv"
	"log"
)

func main() {

	file := &library.WFile{"C:\\__test.txt"}
	str := file.ReadAll()
	//if str != "123" {
		log.Println("ReadAll error: ==>" + str + "<==", len(str))
	//}
	return

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	//library.Reset()
	cpu := runtime.NumCPU()
	log.Println("cpu num: ", cpu)

	//指定cpu为多核运行
	runtime.GOMAXPROCS(cpu)

	current_path := library.GetCurrentPath()
	log.Println(current_path)

	config_file:= current_path+"/config/mysql.ini"
	config_obj := &library.Ini{config_file}
	config     := config_obj.Parse();
	if config == nil {
		log.Println("read config file: " + config_file + " error")
		return
	}
	log.Println(config)

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
		log.Println(err);
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
