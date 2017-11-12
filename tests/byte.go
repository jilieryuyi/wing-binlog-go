package main

import (
    "fmt"
)

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





    as1 := make([]byte, 1024)
    as1 = append(as1[:0], "20123456789"...)

    as1 = append(as1[:0],as1[4:]...)
    fmt.Println(string(as1), len(as1))

    as2 := []byte("20123456789")
    as2 = append(as2[:0],as2[2:2]...)
    fmt.Println(string(as2), len(as2))

    //输出
    //20123456789 1031
    //3456789 7

}
