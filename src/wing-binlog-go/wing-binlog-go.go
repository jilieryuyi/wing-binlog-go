package main

import (
	//"database/sql"
	"library"
	//"github.com/siddontang/go-mysql/canal"

	//"library/base"
	//"library/workers"
	"log"
	_ "github.com/go-sql-driver/mysql"
	"runtime"
	//"strconv"
	//"subscribe"
	"os"
	"os/signal"
	"syscall"
	_"net/http/pprof"
	"net/http"
)

func main() {
	go func() {
		//http://localhost:6060/debug/pprof/  内存性能分析工具
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	wing_log := library.GetLogInstance()
	//释放日志资源
	defer library.FreeLogInstance()

	/*file := &library.WFile{"C:\\__test.txt"}
	str := file.ReadAll()
	//if str != "123" {
		log.Println("ReadAll error: ==>" + str + "<==", len(str))
	//}
	return*/

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	//标准输出重定向
	//library.Reset()
	cpu := runtime.NumCPU()
	wing_log.Println("cpu num: ", cpu)

	//指定cpu为多核运行
	runtime.GOMAXPROCS(cpu)

	current_path := library.GetCurrentPath()
	wing_log.Println(current_path)

	config_file := current_path + "/config/mysql.toml"

	//config_obj := &library.Ini{config_file}
	//config := config_obj.Parse()
	//if config == nil {
	//	wing_log.Println("read config file: " + config_file + " error")
	//	return
	//}
	//wing_log.Println(config)


	//config_file := "/tmp/__test_mysql.toml"
	config := &library.WConfig{config_file}

	app_config, err:= config.Parse()

	if err != nil {
		log.Println(err)
		return
	}


	//user := string(config["mysql"]["user"].(string))
	//password := string(config["mysql"]["password"].(string))
	//port := string(config["mysql"]["port"].(string))
	//host := string(config["mysql"]["host"].(string))

	//slave_id_str := string(config["client"]["slave_id"].(string))
	//slave_id, _ := strconv.Atoi(slave_id_str)

	//db_name := string(config["mysql"]["db_name"].(string))
	//charset := string(config["mysql"]["charset"].(string))
	//db, err := sql.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+db_name+"?charset="+charset)

	//if nil != err {
	//	wing_log.Println(err)
	//	return
	//}

	//defer db.Close()

	blog := library.Binlog{DB_Config:app_config}
	defer blog.Close()
	blog.Start()

	<-sc

	//redis := &subscribe.Redis{}
	//tcp := &subscribe.Tcp{}
    //
	////subscribes
	//notify := []base.Subscribe{redis, tcp}
	//binlog := &workers.Binlog{}
    //
	//defer binlog.End(notify)
    //
	//binlog.Start(notify)
	//binlog.Loop(notify)
}
