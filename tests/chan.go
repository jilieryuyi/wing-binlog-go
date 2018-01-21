package main

import "fmt"

func main() {
	c := make(chan struct{})
	c <- struct{}{}
	 <-c
	c <- struct{}{}
	fmt.Println("main")
}
