package main

import "fmt"

func main() {
	arr := []int{1,2,3,5}
	for i, _:=range arr {
		fmt.Println(i)
	}
}