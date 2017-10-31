package library

import (
    "testing"
)


/**
 * 转换为字符串
 * @return string
 */
func TestToString(t *testing.T)  {
    str := &WString{"hello"}
    if str.ToString() != "hello" {
        t.Error("ToString error - 1")
    }

    str = &WString{[]uint8("123")}
    if str.ToString() != "123" {
        t.Error("ToString error - 2")
    }

    str = &WString{1}
    if str.ToString() != "1" {
        t.Error("ToString error - 3")
    }

    var i64 int64
    i64 = 123456
    str = &WString{i64}
    if str.ToString() != "123456" {
        t.Error("ToString error - 4")
    }

    var iu uint
    iu = 123
    str = &WString{iu}

    if str.ToString() != "123" {
        t.Error("ToString error - 5")
    }
}

/**
 * 截取字符串
 * @param int pos 开始位置
 * @param int length 截取长度
 * @return string
 */
func TestSubstr(t *testing.T)  {
    str := &WString{"1234567"}

    if str.Substr(0, 3) != "123" {
        t.Error("Substr error")
    }

}

/**
 * 获取字符串的长度
 * @return int
 */
func TestLength(t *testing.T)  {
    str := &WString{"123"}

    if str.Length() != 3 {
        t.Error("Length error - 1")
    }

    str = &WString{"你好"}

    if str.Length() != 2 {
        t.Error("Length error - 2")
    }
}


func  TestToInt(t *testing.T) {
    str := &WString{"123"}
    if str.ToInt() != 123 {
        t.Error("ToInt error - 1")
    }

    str = &WString{"123a"}
    if str.ToInt() != 123 {
        t.Error("ToInt error - 2")
    }

    str = &WString{[]uint8("123")}
    if str.ToInt() != 123 {
        t.Error("ToInt error - 3")
    }

    str = &WString{123}
    if str.ToInt() != 123 {
        t.Error("ToInt error - 4")
    }

    var i64 int64 = 123
    str = &WString{i64}
    if str.ToInt() != 123 {
        t.Error("ToInt error - 5")
    }

    var iu uint = 123
    str = &WString{iu}
    if str.ToInt() != 123 {
        t.Error("ToInt error - 6")
    }
}


func TestToInt64(t *testing.T)  {
    str := &WString{"123"}
    if str.ToInt64() != 123 {
        t.Error("ToInt64 error - 1")
    }

    str = &WString{"123.12a"}
    if str.ToInt64() != 123 {
        t.Error("ToInt64 error - 2")
    }

    str = &WString{[]uint8("123")}
    if str.ToInt64() != 123 {
        t.Error("ToInt64 error - 3")
    }

    str = &WString{123}
    if str.ToInt64() != 123 {
        t.Error("ToInt64 error - 4")
    }

    var i64 int64 = 123
    str = &WString{i64}
    if str.ToInt64() != 123 {
        t.Error("ToInt64 error - 5")
    }

    var iu uint = 123
    str = &WString{iu}
    if str.ToInt64() != 123 {
        t.Error("ToInt64 error - 6")
    }
}

func TestToFloat32(t *testing.T)  {
    str := &WString{"123"}
    if str.ToFloat32() != 123 {
        t.Error("ToInt64 error - 1")
    }

    str = &WString{"123.12a"}
    if str.ToFloat32() != 123.12 {
        t.Error("ToInt64 error - 2")
    }

    str = &WString{[]uint8("123")}
    if str.ToFloat32() != 123 {
        t.Error("ToInt64 error - 3")
    }

    str = &WString{123}
    if str.ToFloat32() != 123 {
        t.Error("ToInt64 error - 4")
    }

    var i64 int64 = 123
    str = &WString{i64}
    if str.ToFloat32() != 123 {
        t.Error("ToInt64 error - 5")
    }

    var iu uint = 123
    str = &WString{iu}
    if str.ToFloat32() != 123 {
        t.Error("ToInt64 error - 6")
    }
}

func TestToFloat64(t *testing.T)  {
    str := &WString{"123"}
    if str.ToFloat64() != 123 {
        t.Error("ToInt64 error - 1")
    }

    str = &WString{"123.12a"}
    if str.ToFloat64() != 123.12 {
        t.Error("ToFloat64 error - 2")
    }

    str = &WString{[]uint8("123")}
    if str.ToFloat64() != 123 {
        t.Error("ToFloat64 error - 3")
    }

    str = &WString{123}
    if str.ToFloat64() != 123 {
        t.Error("ToFloat64 error - 4")
    }

    var i64 int64 = 123
    str = &WString{i64}
    if str.ToFloat64() != 123 {
        t.Error("ToFloat64 error - 5")
    }

    var iu uint = 123
    str = &WString{iu}
    if str.ToFloat64() != 123 {
        t.Error("ToFloat64 error - 6")
    }
}



