package main

import "fmt"
const (
    aa = 1 << (iota) //<< 1
    bb
    cc
    dd
)
func main() {
    var a int = 128

    var bin [2]byte
    bin[0] = byte(a)
    bin[1] = byte(a >> 8)

    fmt.Println(bin)

    var b int

    b = int(bin[0]) + int(bin[1] << 8)
    fmt.Println(b)

    fmt.Println(aa,bb, cc, dd)
}
