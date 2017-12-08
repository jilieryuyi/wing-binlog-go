package command

import (
	"library/unix"
)

func Stop() {
	client := unix.NewUnixClient()
	msg := client.Pack(unix.CMD_STOP, "")
	client.Send(msg)
	client.Close()
}
