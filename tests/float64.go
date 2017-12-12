package main

import (
	"fmt"
	//"log"
	"math/big"
)

func main() {
	b := 1.123456789

	f := big.NewFloat(b)
	fmt.Println(f.String())

	fmt.Println(fmt.Sprintf("%f", b))
	var res float64
	res = b
	p := 0
	for i := 0; i < 32; i++ {
		res = res * 10
		p++
		//log.Println(res, float64(int64(res)))
		if (res - float64(int64(res))) == float64(0) {
			break
		}
	}
	fmt.Println(p)
}
