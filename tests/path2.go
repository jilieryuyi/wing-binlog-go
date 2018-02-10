package main

import "fmt"

func main() {
	p := "/usr/a/"
	if p[len(p)-1:] == "/" {
		p=p[:len(p)-1]
	}
	fmt.Println(p)
}
