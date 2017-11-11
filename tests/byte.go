package main

import "fmt"

type S struct {
    buf []byte
}
func main() {
    s := S{}
    s.buf = append(s.buf, "hello"...)
    s.buf = append(s.buf, " world"...)

    //l := len(s.buf)
    //copy(s.buf[l-1:], "1123")

    s2 := make([]byte, len(s.buf)+1)
    copy(s2, s.buf)


    b := s.buf[:0]
    b = append(b, "123"...)
    fmt.Println("==>"+string(s2)+"<==", "==>"+string(s.buf)+"<==", string(b))

    fmt.Println("==" +string([]byte{34})+ "==")


    bbb := make([]byte, 128)
    aaaa:= []byte("sdfsdf")
    ccc := bbb[:2]
    copy(ccc, aaaa)
    fmt.Println("len=", len(ccc), string(ccc) , string(ccc[8:20]))


    var nb [len(ccc)]byte
    copy(nb, aaaa)
    fmt.Println(nb)
}
