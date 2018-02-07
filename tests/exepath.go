package main

import (
	"os"
	"path/filepath"
	"fmt"
)

func main() {
	wd, err := os.Getwd()
	if err == nil {
		fmt.Println(filepath.ToSlash(wd) + "/")
	}
}
