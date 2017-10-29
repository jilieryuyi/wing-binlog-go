package main

import "subscribe/redis"

func main() {
	data := make(map[string] interface{})
	data["hello"] = "yuyi"
	redis.OnChange(data)
}
