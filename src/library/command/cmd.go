package command

import (
	"library/unix"
	log "github.com/sirupsen/logrus"
	"fmt"
)

func Stop() {
	client := unix.NewUnixClient()
	msg := client.Pack(unix.CMD_STOP, "")
	client.Send(msg)
	client.Close()
}

func Reload(cmd string) {
	client := unix.NewUnixClient()
	msg := client.Pack(unix.CMD_RELOAD, cmd)
	client.Send(msg)
	client.Close()
}

func JoinTo(dns string) {
	client := unix.NewUnixClient()
	msg := client.Pack(unix.CMD_JOINTO, dns)
	client.Send(msg)
	client.Close()
}

func ShowMembers() {
	client := unix.NewUnixClient()
	msg := client.Pack(unix.CMD_SHOW_MEMBERS, "")
	client.Send(msg)
	buf, err := client.Read()
	if err != nil {
		log.Errorf("read members error: %+v", err)
	}
	fmt.Println(string(buf))
	client.Close()
}