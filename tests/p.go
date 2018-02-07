package main

import "fmt"

type A struct {
	B string
}

func (a A) s() {
	a.B = "s"
	fmt.Printf("%+v\n", a)
}

func (a *A) s2() {
	a.B = "s2"
}

func main() {
	a := &A{}
	a.s()
	fmt.Printf("%+v\n", a)
	a.s2()
	fmt.Printf("%+v\n", a)
}
