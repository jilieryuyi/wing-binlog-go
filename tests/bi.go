package main

import (
	"fmt"
)

const (
	AgentStatusOffline = 1 << iota //1
	AgentStatusOnline//2
	AgentStatusConnect//4
	AgentStatusDisconnect//8
)
func main() {
	fmt.Println(AgentStatusOffline, AgentStatusOnline, AgentStatusConnect, AgentStatusDisconnect)
	status  := AgentStatusOffline | AgentStatusDisconnect
	//fmt.Println(status & AgentStatusOffline)
	fmt.Println(status)//9

	//fmt.Println(((status & AgentStatusOffline) != 0))

	status ^= AgentStatusOffline
	fmt.Println(status)
	status |= AgentStatusOnline
	fmt.Println(status & AgentStatusOffline)
}
