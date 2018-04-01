package main

import (
	"flag"
	"fmt"
	"library/app"
	"library/binlog"
	_ "github.com/go-sql-driver/mysql"
	"service_plugin/redis"
	"service_plugin/http"
	"service_plugin/tcp"
	"library/control"
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
	cli := control.NewClient(ctx)
	defer cli.Close()
	// 停止服务
	if *stop {
		cli.Stop()
		return true
	}
	// 重新加载服务
	if *serviceReload != "" {
		cli.Reload(*serviceReload)
		return true
	}
	// 帮助
	if *help {
		app.Usage()
		return true
	}
	if *members {
		cli.ShowMembers()
		return true
	}
	return false
}

func main() {
	flag.Parse()
	//defer func() {
	//	if err := recover(); err != nil {
	//		log.Errorf("%+v", err)
	//	}
	//}()
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

	httpService  := http.NewHttpService(appContext)
	tcpService   := tcp.NewTcpService(appContext)
	redisService := redis.NewRedis()

	blog := binlog.NewBinlog(appContext)
	blog.RegisterService(tcpService)
	blog.RegisterService(httpService)
	blog.RegisterService(redisService)
	blog.Start()

	// stop、reload、members ... support
	ctl := control.NewControl(
		appContext,
		control.ShowMember(blog.ShowMembers),
		control.Reload(blog.Reload),
		control.Stop(appContext.Stop),
	)
	ctl.Start()
	defer ctl.Close()

	// wait exit
	select {
		case <- appContext.Done():
	}

	appContext.Cancel()
	blog.Close()
	fmt.Println("wing binlog service exit...")
}
