package main

import "fmt"

func main() {
	var num uint64 = 1
	for i:=0;i < 64; i++ {
		num *= 2
	}
	fmt.Println(1 << 64)
}
