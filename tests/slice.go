package main

import "fmt"

func main() {
	a := []int{1,2,3,4}
	a = append(a[:3], 5)
	fmt.Println(a)
}
