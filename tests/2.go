package main

import "fmt"

func main() {
	a:=1
	b:=2
	c:=4

	d:=a|b

	if d & a > 0 || d & b > 0 {
		fmt.Println("1")
	}

	if d & (a|c) > 0 {
		fmt.Println("2")
	}
}
