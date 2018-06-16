package tcp

import (
	"testing"
	"context"
	"time"
	"bytes"
	"fmt"
	"net"
)

func TestNewServer(t *testing.T) {
	address := "127.0.0.1:7771"
	server  := NewServer(context.Background(), address, SetOnServerMessage(func(node *ClientNode, msgId int64, data []byte) {
		node.Send(msgId, data)
	}))
	server.Start()
	defer server.Close()
	time.Sleep(time.Second)

	go func() {
		dial := net.Dialer{Timeout: time.Second * 3}
		conn, _ := dial.Dial("tcp", address)
		for {
			// 这里发送一堆干扰数据包
			conn.Write([]byte("你好曲儿个人感情如"))
		}
	}()


	client  := NewClient(context.Background())
	err     := client.Connect(address, time.Second * 3)

	if err != nil {
		t.Errorf("connect to %v error: %v", address, err)
		return
	}
	defer client.Disconnect()
	//start := time.Now()
	//times := 1000
	//for  i := 0; i < times; i++ {
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
			t.Errorf("error")
			return
		}
		fmt.Println("w1 return: ", string(res1))
		fmt.Println("w2 return: ", string(res2))
		fmt.Println("w3 return: ", string(res3))
	//}
	//fmt.Println("avg use time ", time.Since(start).Nanoseconds()/int64(times), "ns")
}