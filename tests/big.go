package main

import (
	"math/big"
	"fmt"
	"strconv"
	"github.com/shopspring/decimal"
)

func main() {
	var f float64 = 999999.99999999999999999999999999999999;
	//f = f * (1 << 32)
	fmt.Println((&big.Rat{}).SetFloat64(f).String())
	fmt.Println(strconv.FormatFloat(f, 'f', 32, 64))

	fmt.Println(decimal.NewFromFloat(f).String())
}
