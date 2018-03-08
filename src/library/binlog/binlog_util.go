package binlog

import (
	log "github.com/sirupsen/logrus"
	"github.com/siddontang/go-mysql/schema"
	"reflect"
)

func packPos(binFile string, pos int64, eventIndex int64) []byte {
	res := []byte(binFile)
	l := 16 + len(res)
	r := make([]byte, l + 2)
	// 2 bytes is data length
	r[0] = byte(l)
	r[1] = byte(l >> 8)
	// 8 bytes is pos
	r[2] = byte(pos)
	r[3] = byte(pos >> 8)
	r[4] = byte(pos >> 16)
	r[5] = byte(pos >> 24)
	r[6] = byte(pos >> 32)
	r[7] = byte(pos >> 40)
	r[8] = byte(pos >> 48)
	r[9] = byte(pos >> 56)
	// 8 bytes is event index
	r[10] = byte(eventIndex)
	r[11] = byte(eventIndex >> 8)
	r[12] = byte(eventIndex >> 16)
	r[13] = byte(eventIndex >> 24)
	r[14] = byte(eventIndex >> 32)
	r[15] = byte(eventIndex >> 40)
	r[16] = byte(eventIndex >> 48)
	r[17] = byte(eventIndex >> 56)
	// the last is binlog file
	r = append(r[:18], res...)
	return r
}

func unpackPos(data []byte) (string, int64, int64) {
	dl := int64(data[0]) | int64(data[1]) << 8
	pos := int64(data[2]) | int64(data[3]) << 8 | int64(data[4]) << 16 |
		int64(data[5]) << 24 | int64(data[6]) << 32 | int64(data[7])<<40 |
		int64(data[8]) << 48 | int64(data[9]) << 56
	eventIndex := int64(data[10]) | int64(data[11]) << 8 |
		int64(data[12]) << 16 | int64(data[13]) << 24 |
		int64(data[14]) << 32 | int64(data[15]) << 40 |
		int64(data[16]) << 48 | int64(data[17]) << 56
	if dl+2 < 18 || dl > int64(len(data) - 2) {
		log.Debugf("dl=%d, pos=%d, eventIndex=%d", dl, pos, eventIndex)
		log.Errorf("unpack pos error: %v", data)
		return "", 0, 0
	}
	return string(data[18:dl+2]), pos, eventIndex
}

func fieldDecode(edata interface{}, column *schema.TableColumn) interface{} {
	switch edata.(type) {
	case string:
		return edata
	case []uint8:
		return edata
	case int:
		return edata
	case int8:
		var r int64 = 0
		r = int64(edata.(int8))
		if column.IsUnsigned && r < 0 {
			r = int64(int64(256) + int64(edata.(int8)))
		}
		return r
	case int16:
		var r int64 = 0
		r = int64(edata.(int16))
		if column.IsUnsigned && r < 0 {
			r = int64(int64(65536) + int64(edata.(int16)))
		}
		return r
	case int32:
		var r int64 = 0
		r = int64(edata.(int32))
		if column.IsUnsigned && r < 0 {
			t := string([]byte(column.RawType)[0:3])
			if t != "int" {
				r = int64(int64(1 << 24) + int64(edata.(int32)))
			} else {
				r = int64(int64(4294967296) + int64(edata.(int32)))
			}
		}
		return r
	case int64:
		// 枚举类型支持
		if len(column.RawType) > 4 && column.RawType[0:4] == "enum" {
			i   := int(edata.(int64))-1
			str := column.EnumValues[i]
			return str
		} else if len(column.RawType) > 3 && column.RawType[0:3] == "set" {
			v   := uint(edata.(int64))
			l   := uint(len(column.SetValues))
			res := ""
			for i := uint(0); i < l; i++  {
				if (v & (1 << i)) > 0 {
					if res != "" {
						res += ","
					}
					res += column.SetValues[i]
				}
			}
			return res
		} else {
			if column.IsUnsigned {
				var ur uint64 = 0
				ur = uint64(edata.(int64))
				if ur < 0 {
					ur = 1 << 63 + (1 << 63 + ur)
				}
				return ur
			} else {
				return edata
			}
		}
	case uint:
		return edata
	case uint8:
		return edata
	case uint16:
		return edata
	case uint32:
		return edata
	case uint64:
		return edata
	case float64:
		return edata
	case float32:
		return edata
	default:
		if edata != nil {
			log.Warnf("binlog does not support type：%s %+v", column.Name, reflect.TypeOf(edata))
			return edata
		} else {
			return edata
		}
	}
}

