package main

import (
	"math/big"
	"fmt"
)

func main() {
	var f float64 = 999999.99999999999999999999999999999999;
	fmt.Println(big.NewFloat(f).SetPrec(32).String())
}
