package main

import "fmt"

func main() {
	fmt.Println(fmt.Sprintf("%-25s%s", "service", "isLeader"))
	fmt.Println("---------------------------------")
	fmt.Println(fmt.Sprintf("%-25s%s", "192.168.0.105:9990", "yes"))
}
