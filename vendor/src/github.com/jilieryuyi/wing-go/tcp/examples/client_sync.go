package main

import (
	"github.com/jilieryuyi/wing-go/tcp"
	"fmt"
	"time"
)

type codec struct {}

func (c codec) Encode(msgId int64, msg []byte) []byte {
	return msg
}

func (c codec) Decode(data []byte) (int64, []byte, int, error) {
	return 0, data, len(data), nil
}

func main() {
	//simple http client, try to connect to www.baidu.com
	client := tcp.NewSyncClient(
		"14.215.177.39",
		80,
		tcp.SetWriteTimeout(time.Second * 3),
		tcp.SetReadTimeout(time.Second * 3),
		tcp.SetConnectTimeout(time.Second),
		tcp.SetSyncCoder(&codec{}),
	)
	err := client.Connect()
	if err != nil {
		fmt.Println("error: ", err)
		return
	}
	defer client.Disconnect()
	data := "GET /index.html HTTP/1.1\r\nAccept: text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8\r\nUser-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/66.0.3359.181 Safari/537.36\r\nHost: www.baidu.com\r\nAccept-Language:zh-cn\r\n\r\nhello"
	res, e := client.Send([]byte(data))
	fmt.Println(string(res), e)
}
