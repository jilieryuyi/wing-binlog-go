package unix

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

// -stop
func Stop() {
	client := NewUnixClient()
	msg := client.Pack(CMD_STOP, "")
	client.Send(msg)
	client.Close()
}

//-service-reload http
//-service-reload tcp
//-service-reload all ##重新加载全部服务
//cmd: http、tcp、all
func Reload(cmd string) {
	client := NewUnixClient()
	msg := client.Pack(CMD_RELOAD, cmd)
	client.Send(msg)
	client.Close()
}

// -members
func ShowMembers() {
	client := NewUnixClient()
	msg := client.Pack(CMD_SHOW_MEMBERS, "")
	client.Send(msg)
	buf, err := client.Read()
	if err != nil {
		log.Errorf("read members error: %+v", err)
	}
	fmt.Println(string(buf))
	client.Close()
}
