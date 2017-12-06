package main

import (
	"strings"
	"fmt"
	"strconv"
)
func  getPoint(str string) (int, error) {
	index := strings.IndexByte(str, 44)
	index2 := strings.IndexByte(str, 41)
	return strconv.Atoi(string([]byte(str)[index+1:index2]))
}
func main() {
	str := "decimal(38,2) unsigned"
	sp := strings.Split(str, ",")
	fmt.Println(sp[1])
	sp = strings.Split(sp[1], ")")
	fmt.Println(sp[0])

	fmt.Println([]byte(","), []byte(")"))

	index := strings.IndexByte(str, 44)
	fmt.Println(index)
	index2 := strings.IndexByte(str, 41)
	fmt.Println(index2)

	fmt.Println(string([]byte(str)[index+1:index2]))

	n, _ := getPoint(str)
	fmt.Println("======>", n)

	fmt.Println([]byte("("),[]byte(")"))
	str2 := "enum('F','M')"
	index = strings.IndexByte(str2, 40)
	index2 = strings.IndexByte(str2, 41)
	temp := []byte(str2)[index+1:index2]
	fmt.Println(string(temp))
	arr := strings.Split(string(temp), ",")
	fmt.Println(arr)
	for k, v := range arr {
		arr[k] = string([]byte(v)[1:len(v)-1])
	}
	fmt.Println(arr)
}
