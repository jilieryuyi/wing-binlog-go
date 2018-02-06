package main

import "fmt"

func main() {
	a := []int{1,2,3,4}
	a = append(a[:0], a[4:]...)
	fmt.Println(a)
}
