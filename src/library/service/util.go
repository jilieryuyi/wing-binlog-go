package service

import (
	"regexp"
	"errors"
	//log "github.com/sirupsen/logrus"
)

func MatchFilters(filters []string, table string) bool {
	if filters == nil || len(filters) <= 0 {
		//log.Debugf("主题为空，直接返回true")
		return true
	}
	for _, f := range filters {
		match, err := regexp.MatchString(f, table)
		if match && err == nil {
			return true
		}
	}
	return false
}


func Pack(cmd int, msg []byte) []byte {
	//m  := []byte(msg)
	l  := len(msg)
	r  := make([]byte, l+6)
	cl := l + 2
	r[0] = byte(cl)
	r[1] = byte(cl >> 8)
	r[2] = byte(cl >> 16)
	r[3] = byte(cl >> 24)
	r[4] = byte(cmd)
	r[5] = byte(cmd >> 8)
	copy(r[6:], msg)
	return r
}

var DataLenError = errors.New("data len error")
func Unpack(data []byte) (int, []byte, error) {
	clen := int(data[0]) | int(data[1]) << 8 |
		int(data[2]) << 16 | int(data[3]) << 24
	if len(data) < 	clen + 4 {
		return 0, nil, DataLenError
	}
	cmd  := int(data[4]) | int(data[5]) << 8
	content := data[6 : clen + 4]
	return cmd, content, nil
}