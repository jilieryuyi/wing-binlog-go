package main

import "fmt"

type arr []int

func main() {
	a := []int{3,5,7}
	for v := range a {
		fmt.Println(v)
	}

	a = nil
	fmt.Println(len(a))
	for v := range a {
		fmt.Println(v)
	}

	var dd arr
	dd = append(dd, 1)
	dd = append(dd, 2)
	dd = append(dd, 3)
	fmt.Println("====")
	for v := range dd {
		fmt.Println(v)
	}
}
