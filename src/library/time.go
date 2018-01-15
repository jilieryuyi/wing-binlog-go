package library

import (
	"time"
	"github.com/hashicorp/consul/command/info"
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

// 时间戳格式化为 Y-m-d H:i:s 格式
func Format(ts int64) string {
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

// 将字符串转换为时间戳
// 比如 2018-01-01 08:00:00 转换为时间戳
func ToTimestamp(t string) (int64, error) {
	u, err := time.Parse("2006-01-02 15:04:05", t)
	if err != nil {
		return 0, err
	}
	return u.Unix(), nil
}
