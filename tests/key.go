package main

import (
	"strings"
	"fmt"
)

func main() {
	key := "wing/binlog/keepalive/1516541226-7943-9879-4574"
	i := strings.LastIndex(key, "/")
	fmt.Println(key[i+1:])
}
