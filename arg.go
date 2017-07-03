
//获取命令行参数

package main

import (
	"fmt"
	"os"
)

func main() {
	for key, value := range os.Args {
		fmt.Printf("%d => %s\r\n", key, value);
	}
}
