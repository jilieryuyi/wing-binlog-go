package unix

import (
	"fmt"
	log "github.com/sirupsen/logrus"
)

func Stop() {
	client := NewUnixClient()
	msg := client.Pack(CMD_STOP, "")
	client.Send(msg)
	client.Close()
}

func Reload(cmd string) {
	client := NewUnixClient()
	msg := client.Pack(CMD_RELOAD, cmd)
	client.Send(msg)
	client.Close()
}

func JoinTo(dns string) {
	client := NewUnixClient()
	msg := client.Pack(CMD_JOINTO, dns)
	client.Send(msg)
	client.Close()
}

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

func Clear() {
	client := NewUnixClient()
	msg := client.Pack(CMD_CLEAR, "")
	client.Send(msg)
	_, err := client.Read()
	if err != nil {
		log.Errorf("clear offline members with error: %+v", err)
	}
	client.Close()
}
