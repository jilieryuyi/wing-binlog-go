package main

import (
	"github.com/hashicorp/consul/api"
	"github.com/sirupsen/logrus"
	"time"
	"github.com/jilieryuyi/wing-go/consul"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config := api.DefaultConfig()
	config.Address = "127.0.0.1:8500"

	client, _   := api.NewClient(config)
	serviceName := "service-test2"

	watch := consul.NewWatchService(client.Health(), serviceName)
	go watch.Watch(func(event int, member *consul.ServiceMember) {
		switch event {
		case consul.EventAdd:
			logrus.Infof("%v add service: %+v", time.Now().Unix(), member)
		case consul.EventDelete:
			logrus.Infof("%v delete service: %+v", time.Now().Unix(), member)
		case consul.EventStatusChange:
			logrus.Infof("%v offline service: %+v", time.Now().Unix(), member)
		}
	})

	time.Sleep(time.Second * 3)

	sev := consul.NewService(client.Agent(), serviceName, "127.0.0.1", 7770)
	logrus.Infof("%v register", time.Now().Unix())
	sev.Register()
	defer sev.Deregister()
	a := time.After(time.Second * 35)
	go func() {
		select {
			case <- a:
				sev.Deregister()
		}
	}()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		os.Kill,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	<-sc
}