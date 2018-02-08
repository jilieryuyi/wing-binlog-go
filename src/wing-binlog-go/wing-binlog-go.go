package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"library/app"
	"library/binlog"
	"library/services"
	"library/unix"
	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	"github.com/sevlyar/go-daemon"
	"library/path"
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
	// -deamon === -d run as daemon process
	deamon         = flag.Bool("deamon", false, "-deamon or -d, run as deamon process")
	d              = flag.Bool("d", false, "-deamon or -d, run as deamon process")
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
	if *deamon || *d {
		cntxt := &daemon.Context{
			PidFileName: app.Pid,
			PidFilePerm: 0644,
			LogFileName: path.CurrentPath + "/logs/wing-binlog-go.log",
			LogFilePerm: 0640,
			WorkDir:     path.CurrentPath,
			Umask:       027,
			Args:        []string{"-deamon"},
		}
		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatal("Unable to run: ", err)
		}
		if d != nil {
			return
		}
		defer cntxt.Release()
	}
	// app init
	app.DEBUG = *debug
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

	go func() {
		// wait for exit signal
		sc := make(chan os.Signal, 1)
		signal.Notify(sc,
			os.Kill,
			os.Interrupt,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT)
		<-sc
		appContext.CancelChan <- struct{}{}
	}()

	// wait exit
	select {
		case <- appContext.CancelChan:
	}

	appContext.Cancel()
	blog.Close()
	fmt.Println("wing binlog service exit...")
}
