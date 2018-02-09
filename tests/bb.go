package main

type I interface {
	A() int
}

var (
	_ I = &B{}
)

type B struct{}

func (b *B) A() int {
	return 0
}

func main() {}
