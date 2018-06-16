package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"time"
	"fmt"
	"github.com/sirupsen/logrus"
	"bytes"
	"net/http"
	_ "net/http/pprof"
)

func main() {
	// 先运行server端
	// go run server.go
	// 在运行client端
	// go run client
	address := "127.0.0.1:7771"
	client  := tcp.NewClient(context.Background())
	err     := client.Connect(address, time.Second * 3)

	if err != nil {
		logrus.Panicf("connect to %v error: %v", address, err)
		return
	}
	defer client.Disconnect()

	go func() {
		//http://localhost:8880/debug/pprof/  内存性能分析工具
		//go tool pprof logDemo.exe --text a.prof
		//go tool pprof your-executable-name profile-filename
		//go tool pprof your-executable-name http://localhost:8880/debug/pprof/heap
		//go tool pprof wing-binlog-go http://localhost:8880/debug/pprof/heap
		//https://lrita.github.io/2017/05/26/golang-memory-pprof/
		//然后执行 text
		//go tool pprof -alloc_space http://127.0.0.1:8880/debug/pprof/heap
		//top20 -cum

		//下载文件 http://localhost:8880/debug/pprof/profile
		//分析 go tool pprof -web /Users/yuyi/Downloads/profile
		http.ListenAndServe("127.0.0.1:7773", nil)
	}()

	start := time.Now()
	times := 1000000
	for  {
		data1 := []byte("hello")
		data2 := []byte("word")
		data3 := []byte("hahahahahahahahahahah")
		w1, _, _ := client.Send(data1)
		w2, _, _ := client.Send(data2)
		w3, _, _ := client.Send(data3)
		res1, _ := w1.Wait(time.Second * 3)
		res2, _ := w2.Wait(time.Second * 3)
		res3, _ := w3.Wait(time.Second * 3)

		if !bytes.Equal(data1, res1) || !bytes.Equal(data2, res2) || !bytes.Equal(data3, res3) {
			logrus.Panicf("error")
		}
		fmt.Println("w1 return: ", string(res1))
		fmt.Println("w2 return: ", string(res2))
		fmt.Println("w3 return: ", string(res3))
	}
	fmt.Println("avg use time ", time.Since(start).Nanoseconds()/int64(times)/1000000, "ms")
}
