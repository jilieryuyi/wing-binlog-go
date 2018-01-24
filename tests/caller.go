package main

import "time"

func main() {
	go a()
	time.Sleep(time.Second*10000)
}
