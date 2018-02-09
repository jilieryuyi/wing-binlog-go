package string

import (
	"math/rand"
	"time"
)

func RandString(slen int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bt := []byte(str)
	result := make([]byte, 0)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < slen; i++ {
		result = append(result, bt[r.Intn(len(bt))])
	}
	return string(result)
}