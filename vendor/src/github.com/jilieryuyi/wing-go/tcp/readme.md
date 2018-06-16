the easy way to use go tcp

demo server
````
package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"os"
	"os/signal"
	"syscall"
)
func main() {
	address := "127.0.0.1:7770"
	server  := tcp.NewAgentServer(context.Background(), address, tcp.SetOnServerMessage(func(node *tcp.ClientNode, msgId int64, data []byte) {
		node.Send(msgId, data)
	}))
	server.Start()
	defer server.Close()

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
````

demo client
````
package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"context"
	"time"
	"fmt"
	"github.com/sirupsen/logrus"
	"bytes"
)
func main() {
	// 先运行server端
	// go run server.go
	// 在运行client端
	// go run client
	address := "127.0.0.1:7770"
	client  := tcp.NewClient(context.Background())
	err     := client.Connect(address, time.Second * 3)

	if err != nil {
		logrus.Panicf("connect to %v error: %v", address, err)
		return
	}
	defer client.Disconnect()

    data1 := []byte("hello")
    data2 := []byte("word")
    data3 := []byte("hahahahahahahahahahah")
    w1, _ := client.Send(data1)
    w2, _ := client.Send(data2)
    w3, _ := client.Send(data3)
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

````
