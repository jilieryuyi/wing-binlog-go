package main

import (
    "strconv"
    "fmt"
)

func main() {
    f := 100.12345678901234567890123456789
    b := make([]byte, 0)
    b = strconv.AppendFloat(b, f, 'f', 5, 64)

    fmt.Println(string(b))
}
