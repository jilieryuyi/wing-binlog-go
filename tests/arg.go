package main

import (
	"fmt"
	"os"
)

func main() {
	//    args := os.Args
	for key, value := range os.Args {
		fmt.Printf("%d => %s\r\n", key, value)
	}
}
