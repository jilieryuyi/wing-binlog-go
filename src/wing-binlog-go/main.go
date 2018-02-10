package main

import (
	"flag"
	"fmt"
	"library/app"
	"library/binlog"
	"library/services"
	"library/unix"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

var (
	//if debug is true, print stack log
	debug   = flag.Bool("debug", false, "enable debug, default disable")
	version = flag.Bool("version", false, "wing binlog go version")
	stop    = flag.Bool("stop", false, "stop service")
	//-service-reload http
	//-service-reload tcp
	//-service-reload all ##重新加载全部服务
	serviceReload  = flag.String("service-reload", "", "reload service config, usage: -service-reload  all|http|tcp")
	// show help info
	help           = flag.Bool("help", false, "help")
	// show members
	members        = flag.Bool("members", false, "show members from current node")
	// -daemon === -d run as daemon process
	daemon         = flag.Bool("daemon", false, "-daemon or -d, run as daemon process")
	d              = flag.Bool("d", false, "-daemon or -d, run as daemon process")
	configPath     = flag.String("config-path", "", "-config-path set config path, default is ./config")
)

func Cmd() bool {
	// 显示版本信息
	if *version {
		fmt.Println(app.VERSION)
		return true
	}
	// 停止服务
	if *stop {
		unix.Stop()
		return true
	}
	// 重新加载服务
	if *serviceReload != "" {
		unix.Reload(*serviceReload)
		return true
	}
	// 帮助
	if *help {
		app.Usage()
		return true
	}
	if *members {
		unix.ShowMembers()
		return true
	}
	return false
}

func main() {
	flag.Parse()
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("%+v", err)
		}
	}()
	// if use cmd params
	if Cmd() {
		return
	}
	// return true is parent process
	if app.DaemonProcess(*daemon || *d) {
		return
	}
	// app init
	app.DEBUG = *debug
	app.ConfigPathParse(*configPath)
	app.Init()
	// clear some resource after exit
	defer app.Release()

	appContext  := app.NewContext()
	httpService := services.NewHttpService(appContext)
	tcpService  := services.NewTcpService(appContext)

	blog := binlog.NewBinlog(appContext)
	blog.RegisterService("tcp", tcpService)
	blog.RegisterService("http", httpService)
	blog.Start()

	// unix socket use for cmd support
	server := unix.NewUnixServer(appContext, blog)
	server.Start()
	defer server.Close()

	// wait exit
	select {
		case <- appContext.CancelChan:
	}

	appContext.Cancel()
	blog.Close()
	fmt.Println("wing binlog service exit...")
}
