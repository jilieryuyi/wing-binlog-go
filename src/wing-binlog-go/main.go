package main

import (
	"flag"
	"fmt"
	"library/app"
	"library/binlog"
	_ "github.com/go-sql-driver/mysql"
	"services/redis"
	"services/http"
	"services/kafka"
	"library/control"
	"library/agent"
	log "github.com/sirupsen/logrus"
	"services/subscribe"
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
	// 这里初始化一些全局配置信息
	app.DEBUG = *debug
	app.Init(isCmd, *configPath)

	// clear some resource after exit
	defer app.Release()
	appContext  := app.NewContext()

	// if use cmd params
	// 命令支持
	// -stop
	// -reload
	// ... 详细信息可以使用 -help 帮助
	// demo: ./bin/wing-binlog-go -help
	if isCmd {
		runCmd(appContext)
		return
	}
	// return true is parent process
	// 如果返回true，代表已守护进程运行
	// 启动的时候带了 -d 或者 -daemon 参数
	if app.DaemonProcess(*daemon || *d) {
		return
	}

	// 四个服务插件 http、redis、kafka、tcp（subscribe）
	httpService      := http.NewHttpService(appContext)
	redisService     := redis.NewRedis()
	kafkaService     := kafka.NewProducer()
	subscribeService := subscribe.NewSubscribeService(appContext)

	// agent代理，用于实现集群
	agentServer := agent.NewAgentServer(
		appContext,
		agent.OnEvent(subscribeService.SendAll),
		agent.OnRaw(subscribeService.SendRaw),
	)

	// 核心binlog服务
	blog := binlog.NewBinlog(
		appContext,
		binlog.PosChange(agentServer.SendPos),
		binlog.OnEvent(agentServer.SendEvent),
	)
	// 注册服务
	blog.RegisterService(httpService)
	blog.RegisterService(redisService)
	blog.RegisterService(kafkaService)
	blog.RegisterService(subscribeService)
	// 开始binlog进程
	blog.Start()

	// set agent receive pos callback
	// 延迟依赖绑定
	// agent与binlog相互依赖
	// agent收到leader的pos改变同步信息时，回调到SaveBinlogPosition
	// agent选leader成功回调到OnLeader上，是为了停止和开启服务，只有leader在工作
	agent.OnPos(blog.SaveBinlogPosition)(agentServer)
	agent.OnLeader(blog.OnLeader)(agentServer)

	// 启动agent进程
	agentServer.Start()
	defer agentServer.Close()

	// 热更新reload支持
	var reload = func(name string) {
		if name == "all" {
			redisService.Reload()
			httpService.Reload()
			kafkaService.Reload()
			subscribeService.Reload()
		} else {
			switch name {
			case httpService.Name():
				httpService.Reload()
			case redisService.Name():
				redisService.Reload()
			case subscribeService.Name():
				subscribeService.Reload()
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
