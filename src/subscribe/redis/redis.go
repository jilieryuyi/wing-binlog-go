package redis

import "fmt"

func OnChange(data map[string] interface{}) {
	fmt.Println(data)
}

