package String

import (
	"fmt"
	"reflect"
	"strconv"

)

type Convert struct {
	Data interface {}
}

const WING_DEBUG = true


func (r Convert) ToString() string {
	fmt.Printf("类型是：%s\r\n", reflect.TypeOf(r.Data))
	switch r.Data.(type) {
	case string:
		return string(r.Data.(string));
	case []uint8:
		return string(r.Data.([]byte))
	case int:
		return strconv.Itoa(int(r.Data.(int)))
	case int64:
		return strconv.FormatInt(int64(r.Data.(int64)),10)
	case uint:
		return strconv.Itoa(int(r.Data.(uint)))
	}
	return "";
}


func (r Convert) toInt() int {
	fmt.Printf("类型是：%s\r\n", reflect.TypeOf(r.Data))
	var d int = 0
	switch r.Data.(type) {
	case string:
		d, _ = strconv.Atoi(string(r.Data.(string)))
		return d;
	case []uint8:
		d, _ = strconv.Atoi( string(r.Data.([]byte)))
		return d
	case int:
		return int(r.Data.(int))
	case int64:
		return int(r.Data.(int64))
	case uint:
		return int(r.Data.(uint))
	}
	return 0;
}


//func main() {
//	var v int64 = 123456
//	r := Convert{v};
//	s := r.toString();
//	fmt.Println("hello"+s);
//
//	i := r.toInt()
//	fmt.Println(i+10);
//}