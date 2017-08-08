package main

import (
	"fmt"
	"reflect"
	"strconv"
)

type Convert struct {
	data interface {}
}


func (r Convert) toString() string {
	fmt.Printf("类型是：%s\r\n", reflect.TypeOf(r.data))
	switch r.data.(type) {
	case string:
		return string(r.data.(string));
	case []uint8:
		return string(r.data.([]byte))
	case int:
		return strconv.Itoa(int(r.data.(int)))
	case int64:
		return strconv.FormatInt(int64(r.data.(int64)),10)
	case uint:
		return strconv.Itoa(int(r.data.(uint)))
	}
	return "";
}


func (r Convert) toInt() int {
	fmt.Printf("类型是：%s\r\n", reflect.TypeOf(r.data))
	var d int = 0
	switch r.data.(type) {
	case string:
		d, _ = strconv.Atoi(string(r.data.(string)))
		return d;
	case []uint8:
		d, _ = strconv.Atoi( string(r.data.([]byte)))
		return d
	case int:
		return int(r.data.(int))
	case int64:
		return int(r.data.(int64))
	case uint:
		return int(r.data.(uint))
	}
	return 0;
}


func main() {
	var v uint = 1234
	r := Convert{v};
	s := r.toString();
	fmt.Println("hello"+s);

	i := r.toInt()
	fmt.Println(i+10);
}