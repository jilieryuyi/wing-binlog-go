package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	wclient "github.com/jilieryuyi/wing-binlog-go/src/library/client"
)
// 执行 go get github.com/jilieryuyi/wing-binlog-go
// 安装wing-binlog-go依赖
func main() {
	//初始化debug终端输出日志支持
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.SetLevel(log.Level(5))

	// event callback
	// 有事件过来的时候，就会进入这个回调
	var onEvent = func(data map[string]interface{}) {
		fmt.Printf("new event: %+v", data)
	}

	// 创建一个新的客户端
	// 第一个参数为一个数组，这里可以指定多个地址，当其中一个失败的时候自动轮训下一个
	// 简单的高可用方案支持
	// 第二个参数为注册事件回调

	//defaultDns := "127.0.0.1:9996"
	//if len(os.Args) >= 2 {
	//	defaultDns = os.Args[1]
	//}
	//client := wclient.NewClient(wclient.SetServices([]string{defaultDns}), wclient.OnEventOption(onEvent))

	//或者使用consul
	client := wclient.NewClient(wclient.SetConsulAddress("127.0.0.1:8500"), wclient.OnEventOption(onEvent))

	// 程序退出时 close 掉客户端
	defer client.Close()
	//订阅感兴趣的数据库、表变化事件
	//如果不订阅，默认对所有的变化感兴趣
	//client.Subscribe("new_yonglibao_c.*", "test.*")

	// 等待退出信号，比如control+c
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt)
	<-signals
}