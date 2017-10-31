package library

import (
	//"fmt"
	//"reflect"
	"strconv"
)

type WString struct {
	Str interface {}
}

/**
 * 转换为字符串
 * @return string
 */
func (str *WString) ToString() string {
	//fmt.Printf("类型是：%s\r\n", reflect.TypeOf(str.Data))
	switch str.Str.(type) {
	case string:
		return str.Str;//string(str.Str.(string));
	case []uint8:
		return string(str.Str.([]byte))
	case int:
		return strconv.Itoa(int(str.Str.(int)))
	case int64:
		return strconv.FormatInt(int64(str.Str.(int64)),10)
	case uint:
		return strconv.Itoa(int(str.Str.(uint)))
	}
	return "";
}

/**
 * 截取字符串
 * @param int pos 开始位置
 * @param int length 截取长度
 * @return string
 */
func (str *WString) Substr(pos int, length int) string {
	runes := []rune(str.ToString())
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

/**
 * 获取字符串的长度
 * @return int
 */
func (str *WString) Length() int {
	return len([]rune(str.ToString()))
}


func (str *WString) toInt() int {
	//fmt.Printf("类型是：%s\r\n", reflect.TypeOf(r.Data))
	var d int = 0
	switch str.Str.(type) {
	case string:
		d, _ = strconv.Atoi(string(str.Str.(string)))
		return d;
	case []uint8:
		d, _ = strconv.Atoi( string(str.Str.([]byte)))
		return d
	case int:
		return int(str.Str.(int))
	case int64:
		return int(str.Str.(int64))
	case uint:
		return int(str.Str.(uint))
	}
	return 0;
}


func (str *WString) toInt64() int64 {
	//fmt.Printf("类型是：%s\r\n", reflect.TypeOf(r.Data))
	var d int64 = 0
	switch str.Str.(type) {
	case string:
		d, _ = strconv.ParseInt(string(str.Str.(string)), 10, 0)
		return d;
	case []uint8:
		d, _ = strconv.ParseInt(string(str.Str.([]byte)), 10, 0)
		return d
	case int:
		return int64(str.Str.(int))
	case int64:
		return int64(str.Str.(int64))
	case uint:
		return int64(str.Str.(uint))
	}
	return 0;
}
