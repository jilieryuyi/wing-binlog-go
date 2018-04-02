package app

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"syscall"
	"context"
)
// context
type Context struct {
	// canal context
	Ctx context.Context
	// canal context func
	Cancel context.CancelFunc
	// pid file path
	PidFile string
	cancelChan chan struct{}
	//reloadChan chan string
	//ShowMembersChan chan struct{}
	//ShowMembersRes chan string
	PosChan chan string
	//HttpConfig *HttpConfig
	TcpConfig *TcpConfig
	MysqlConfig *MysqlConfig
	ClusterConfig *ClusterConfig
	AppConfig *Config
}

// new app context
func NewContext() *Context {
	//httpConfig, _ := getHttpConfig()
	tcpConfig, _ := getTcpConfig()
	mysqlConfig, _:= getMysqlConfig()
	clusterConfig, _ := getClusterConfig()
	appConfig, _ := getAppConfig()
	ctx := &Context{
		cancelChan:make(chan struct{}),
		//reloadChan:make(chan string, 100),
		//ShowMembersChan:make(chan struct{}, 100),
		//ShowMembersRes:make(chan string, 12),
		//PosChan:make(chan string, 10000),
		//HttpConfig: httpConfig,
		TcpConfig: tcpConfig,
		MysqlConfig: mysqlConfig,
		ClusterConfig: clusterConfig,
		AppConfig:appConfig,
	}
	ctx.Ctx, ctx.Cancel = context.WithCancel(context.Background())
	go ctx.signalHandler()
	return ctx
}


func (ctx *Context) Stop() {
	ctx.cancelChan <- struct{}{}
}

func (ctx *Context) Done() <-chan struct{} {
	return ctx.cancelChan
}

//func (ctx *Context) Reload(serviceName string) {
//	ctx.reloadChan <- serviceName
//}

//func (ctx *Context) ReloadDone() <-chan string {
//	return ctx.reloadChan
//}

//func (ctx *Context) ReloadHttpConfig() {
//	httpConfig, err := getHttpConfig()
//	if err != nil {
//		log.Errorf("get http config error: %v", err)
//		return
//	}
//	ctx.HttpConfig = httpConfig
//}

func (ctx *Context) ReloadTcpConfig() {
	tcpConfig, err := getTcpConfig()
	if err != nil {
		log.Errorf("get tcp config error: %v", err)
		return
	}
	ctx.TcpConfig = tcpConfig
}

// wait for control + c signal
func (ctx *Context) signalHandler() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sc
	log.Warnf("get exit signal, service will exit later")
	ctx.cancelChan <- struct{}{}
}
