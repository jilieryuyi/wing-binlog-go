package main

import (
    //"reflect"
    "fmt"
    "strconv"
)

func main() {


    var str interface{} = "123.12a"
    data  := string(str.(string))
    res   := ""
    start := false

    var d float32 = 0.0

    for k, v := range data {
        if (k == 0) {
            if (v < 48 || v > 57) {
                break
            }
        }
        if ((v >= 48 && v <= 57) || v == 46) {
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

    d, _ = strconv.ParseInt(res, 10, 0)
    return d
}
