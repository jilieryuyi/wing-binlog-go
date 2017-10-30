package library

import (
	"time"
)

func GetDayTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func GetDay(format string) string {
	return time.Now().Format(format)
}

func GetTimeStamp() int64 {
	return time.Now().Unix()
}