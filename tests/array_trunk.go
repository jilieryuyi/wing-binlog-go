package main

import (
    //"math"
    "fmt"
)

func array_trunk(m map[string] int, n int) []map[string] int {
    l := len(m)
    s := int(l/n)

    if l%n != 0 {
        s += 1
    }


    fmt.Println(s)
    res := make([]map[string] int, s)


    for i :=0; i < s; i++ {
        res[i] = make(map[string] int)
    }
    index := 0
    sk := 0
    for k, _ := range m {
        fmt.Println(index, k,m[k])
        res[index][k] = m[k]
        sk++
        if sk == n {
            index++
            sk=0
        }
    }
    return res
}

func main() {
    m := make(map[string] int)
    m["1"] = 1
    m["2"] = 2
    m["3"] = 3
    m["4"] = 4
    m["5"] = 5

    nm := array_trunk(m, 2)
    fmt.Println(nm)
}
