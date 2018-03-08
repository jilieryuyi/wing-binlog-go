package main

import "fmt"

const (
	//start stop
	_binlogIsRunning = 1 << iota
	// binlog is in exit status, will exit later
	_binlogIsExit
	_cacheHandlerISOpened
	_consulIsLeader
	_enableConsul
)

func main() {
	fmt.Println(_binlogIsRunning)
}