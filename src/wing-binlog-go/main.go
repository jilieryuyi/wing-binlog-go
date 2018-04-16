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
	"service_plugin/kafka"
	"library/control"
	"library/agent"
	log "github.com/sirupsen/logrus"
	"service_plugin/subscribe"
)

var (
	//if debug is true, print stack log
	debug   = flag.Bool("debug", false, "enable debug, default disable")
	version = flag.Bool("version", false, "wing binlog go version")
	v = flag.Bool("v", false, "wing binlog go version")
	stop    = flag.Bool("stop", false, "stop service")
	//-service-reload http
	//-service-reload tcp
	//-service-reload all ##重新加载全部服务
	serviceReload  = flag.String("reload", "", "reload service config, usage: -reload  all|http|tcp|redis")
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
	if *version || *v {
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
	isCmd := *version || *stop || *serviceReload != "" || *help || *members || *v
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

	httpService      := http.NewHttpService(appContext)
	tcpService       := tcp.NewTcpService(appContext)
	redisService     := redis.NewRedis()
	kafkaService     := kafka.NewProducer()
	subscribeService := subscribe.NewSubscribeService(appContext)

	agentServer := agent.NewAgentServer(
		appContext,
		agent.OnEvent(tcpService.SendAll),
		agent.OnRaw(tcpService.SendRaw),
	)

	blog := binlog.NewBinlog(
		appContext,
		binlog.PosChange(agentServer.SendPos),
		binlog.OnEvent(agentServer.SendEvent),
	)
	blog.RegisterService(tcpService)
	blog.RegisterService(httpService)
	blog.RegisterService(redisService)
	blog.RegisterService(kafkaService)
	blog.RegisterService(subscribeService)
	blog.Start()

	// set agent receive pos callback
	// 延迟依赖绑定
	// agent与binlog相互依赖
	agent.OnPos(blog.SaveBinlogPosition)(agentServer)
	agent.OnLeader(blog.OnLeader)(agentServer)

	agentServer.Start()
	defer agentServer.Close()

	var reload = func(name string) {
		if name == "all" {
			tcpService.Reload()
			redisService.Reload()
			httpService.Reload()
			kafkaService.Reload()
		} else {
			switch name {
			case httpService.Name():
				httpService.Reload()
			case tcpService.Name():
				tcpService.Reload()
			case redisService.Name():
				redisService.Reload()
			default:
				log.Errorf("unknown service: %v", name)
			}
		}
	}
	// stop、reload、members ... support
	// 本地控制命令支持
	ctl := control.NewControl(
		appContext,
		control.ShowMember(agentServer.ShowMembers),
		control.Reload(reload),
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
