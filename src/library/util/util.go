package util

import (
	"time"
	"math/rand"
)

func RandString() string {
	str    := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bt     := []byte(str)
	result := []byte{}
	r      := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 32; i++ {
		result = append(result, bt[r.Intn(len(bt))])
	}

	return string(result)
}
