package main

import "fmt"

const cc = 2
const (
	a uint16 = 1 << iota // + 0xf5//iota//cc + 1// << iota //byte = iota + 1//
	b
	c
	d
)

func main() {
	fmt.Println(a, b, c, d)
}
