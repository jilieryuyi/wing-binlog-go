package string

import (
	"strconv"
)

type WString struct {
	Str interface{}
}

/**
 * 转换为字符串
 * @return string
 */
func (str *WString) ToString() string {
	//fmt.Println("type======", str.Str, "=>", reflect.TypeOf(str.Str))
	switch str.Str.(type) {
	case string:
		return string(str.Str.(string))
	case []uint8:
		return string(str.Str.([]byte))
	case int:
		return strconv.Itoa(int(str.Str.(int)))
	case int64:
		return strconv.FormatInt(int64(str.Str.(int64)), 10)
	case uint:
		return strconv.Itoa(int(str.Str.(uint)))
	}
	return ""
}

/**
 * 截取字符串
 * @param int pos 开始位置，如果是负数，从尾部开始截取，比如-2代表倒数第二个字符
 * @param int length 截取长度
 * @return string
 */
func (str *WString) Substr(pos int, length int) string {
	runes := []rune(str.ToString())
	if pos < 0 {
		pos = len(runes) + pos
	}

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

func (str *WString) ToInt() int {
	switch str.Str.(type) {
	case string:
		data := []byte(string(str.Str.(string)))
		res := ""
		start := false

		for _, v := range data {
			if v >= 48 && v <= 57 {
				res += string(v)
				start = true
			} else {
				if start {
					break
				}
			}
		}

		if res == "" {
			res = "0"
		}

		d, _ := strconv.Atoi(res)
		return d
	case []uint8:
		d, _ := strconv.Atoi(string(str.Str.([]byte)))
		return d
	case int:
		return int(str.Str.(int))
	case int64:
		return int(str.Str.(int64))
	case uint:
		return int(str.Str.(uint))
	}
	return 0
}

func (str *WString) ToInt64() int64 {
	switch str.Str.(type) {
	case string:
		data := string(str.Str.(string))
		res := ""
		start := false

		for _, v := range data {
			if v >= 48 && v <= 57 {
				res += string(v)
				start = true
			} else {
				if start {
					break
				}
			}
		}

		if res == "" {
			res = "0"
		}

		d, _ := strconv.ParseInt(res, 10, 0)
		return d
	case []uint8:
		d, _ := strconv.ParseInt(string(str.Str.([]byte)), 10, 0)
		return d
	case int:
		return int64(str.Str.(int))
	case int64:
		return int64(str.Str.(int64))
	case uint:
		return int64(str.Str.(uint))
	}
	return 0
}

func (str *WString) ToFloat32() float32 {
	switch str.Str.(type) {
	case string:
		data := string(str.Str.(string))
		res := ""
		start := false

		for k, v := range data {
			if k == 0 {
				if v < 48 || v > 57 {
					break
				}
			}
			if (v >= 48 && v <= 57) || v == 46 {
				res += string(v)
				start = true
			} else {
				if start {
					break
				}
			}
		}

		if res == "" {
			res = "0"
		}

		d1, _ := strconv.ParseFloat(res, 32)
		return float32(d1)
	case []uint8:
		d1, _ := strconv.ParseFloat(string(str.Str.([]byte)), 32)
		return float32(d1)
	case int:
		return float32(str.Str.(int))
	case int64:
		return float32(str.Str.(int64))
	case uint:
		return float32(str.Str.(uint))
	}
	return 0
}

func (str *WString) ToFloat64() float64 {
	switch str.Str.(type) {
	case string:
		data := string(str.Str.(string))
		res := ""
		start := false

		for k, v := range data {
			if k == 0 {
				if v < 48 || v > 57 {
					break
				}
			}
			if (v >= 48 && v <= 57) || v == 46 {
				res += string(v)
				start = true
			} else {
				if start {
					break
				}
			}
		}

		if res == "" {
			res = "0"
		}

		d1, _ := strconv.ParseFloat(res, 64)
		return d1
	case []uint8:
		d1, _ := strconv.ParseFloat(string(str.Str.([]byte)), 64)
		return d1
	case int:
		return float64(str.Str.(int))
	case int64:
		return float64(str.Str.(int64))
	case uint:
		return float64(str.Str.(uint))
	}
	return 0
}
