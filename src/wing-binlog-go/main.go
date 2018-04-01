package main

import (
	"flag"
	"fmt"
	"library/app"
	"library/binlog"
	"library/services"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"service_plugin"
)

var (
	//if debug is true, print stack log
	debug   = flag.Bool("debug", false, "enable debug, default disable")
	version = flag.Bool("version", false, "wing binlog go version")
	stop    = flag.Bool("stop", false, "stop service")
	//-service-reload http
	//-service-reload tcp
	//-service-reload all ##重新加载全部服务
	serviceReload  = flag.String("reload", "", "reload service config, usage: -reload  all|http|tcp")
	// show help info
	help           = flag.Bool("help", false, "help")
	// show members
	members        = flag.Bool("members", false, "show members from current node")
	// -daemon === -d run as daemon process
	daemon         = flag.Bool("daemon", false, "-daemon or -d, run as daemon process")
	d              = flag.Bool("d", false, "-daemon or -d, run as daemon process")
	configPath     = flag.String("config-path", "", "-config-path set config path, default is ./config")
)

func runCmd(ctx *app.Context) bool {
	// 显示版本信息
	if *version {
		fmt.Println(app.VERSION)
		return true
	}
	control := services.NewControl(ctx)
	defer control.Close()
	// 停止服务
	if *stop {
		control.Stop()
		return true
	}
	// 重新加载服务
	if *serviceReload != "" {
		control.Reload(*serviceReload)
		return true
	}
	// 帮助
	if *help {
		app.Usage()
		return true
	}
	if *members {
		control.ShowMembers()
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
	isCmd := *version || *stop || *serviceReload != "" || *help || *members
	// app init
	app.DEBUG = *debug
	app.Init(isCmd, *configPath)

	// clear some resource after exit
	defer app.Release()
	appContext  := app.NewContext()

	// if use cmd params
	if isCmd {
		runCmd(appContext)
		return
	}
	// return true is parent process
	if app.DaemonProcess(*daemon || *d) {
		return
	}

	httpService := services.NewHttpService(appContext)
	tcpService  := services.NewTcpService(appContext)

	blog := binlog.NewBinlog(appContext)
	blog.RegisterService(binlog.ServiceNameTcp, tcpService)
	blog.RegisterService(binlog.ServiceNameHttp, httpService)
	blog.RegisterService("redis", service_plugin.NewRedis())
	blog.Start()

	// wait exit
	select {
		case <- appContext.Done():
	}

	appContext.Cancel()
	blog.Close()
	fmt.Println("wing binlog service exit...")
}
