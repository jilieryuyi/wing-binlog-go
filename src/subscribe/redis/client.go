package redis

import "fmt"

type Redis struct {

}

func (r *Redis) OnChange(data map[string] interface{}) {
	fmt.Println("redis", data)
}

