package main

import "fmt"

const (
	a = iota
	b
	c
	d = 9
	e = 1 << iota
	f
)

func main() {
	fmt.Println(a,b,c,d, e,f)
}