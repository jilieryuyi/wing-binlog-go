package binlog

import (
	"math/big"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	"github.com/siddontang/go-mysql/schema"
	log "github.com/sirupsen/logrus"
)

func (h *binlogHandler) RegisterService(name string, s services.Service) {
	h.services[name] = s
	h.servicesCount++
}

func (h *binlogHandler) notify(msg []byte) {
	log.Debug("binlog发送广播：", msg, string(msg))
	for _, service := range h.services {
		service.SendAll(msg)
	}
}

func (h *binlogHandler) append(buf *[]byte, edata interface{}, column *schema.TableColumn) {
	//log.Debugf("%+v,===,%+v, == %+v", column, reflect.TypeOf(edata), edata)
	switch edata.(type) {
	case string:
		encode(buf, edata.(string))
	case []uint8:
		encode(buf, string(edata.([]byte)))
	case int:
		*buf = strconv.AppendInt(*buf, int64(edata.(int)), 10)
	case int8:
		var r int64 = 0
		r = int64(edata.(int8))
		if column.IsUnsigned && r < 0 {
			r = int64(int64(256) + int64(edata.(int8)))
		}
		*buf = strconv.AppendInt(*buf, r, 10)
	case int16:
		var r int64 = 0
		r = int64(edata.(int16))
		if column.IsUnsigned && r < 0 {
			r = int64(int64(65536) + int64(edata.(int16)))
		}
		*buf = strconv.AppendInt(*buf, r, 10)
	case int32:
		var r int64 = 0
		r = int64(edata.(int32))
		if column.IsUnsigned && r < 0 {
			t := string([]byte(column.RawType)[0:3])
			if t != "int" {
				r = int64(int64(1<<24) + int64(edata.(int32)))
			} else {
				r = int64(int64(4294967296) + int64(edata.(int32)))
			}
		}
		*buf = strconv.AppendInt(*buf, r, 10)
	case int64:
		// 枚举类型支持
		if len(column.RawType) > 4 && column.RawType[0:4] == "enum" {
			i := int(edata.(int64)) - 1
			str := column.EnumValues[i]
			encode(buf, str)
		} else if len(column.RawType) > 3 && column.RawType[0:3] == "set" {
			v := uint(edata.(int64))
			l := uint(len(column.SetValues))
			res := ""
			for i := uint(0); i < l; i++ {
				if (v & (1 << i)) > 0 {
					if res != "" {
						res += ","
					}
					res += column.SetValues[i]
				}
			}
			encode(buf, res)
		} else {
			if column.IsUnsigned {
				var ur uint64 = 0
				ur = uint64(edata.(int64))
				if ur < 0 {
					ur = 1<<63 + (1<<63 + ur)
				}
				*buf = strconv.AppendUint(*buf, ur, 10)
			} else {
				*buf = strconv.AppendInt(*buf, int64(edata.(int64)), 10)
			}
		}
	case uint:
		*buf = strconv.AppendUint(*buf, uint64(edata.(uint)), 10)
	case uint8:
		*buf = strconv.AppendUint(*buf, uint64(edata.(uint8)), 10)
	case uint16:
		*buf = strconv.AppendUint(*buf, uint64(edata.(uint16)), 10)
	case uint32:
		*buf = strconv.AppendUint(*buf, uint64(edata.(uint32)), 10)
	case uint64:
		*buf = strconv.AppendUint(*buf, uint64(edata.(uint64)), 10)
	case float64:
		f := big.NewFloat(edata.(float64))
		*buf = append(*buf, f.String()...)
	case float32:
		f := big.NewFloat(float64(edata.(float32)))
		*buf = append(*buf, f.String()...)
	default:
		if edata != nil {
			log.Warnf("binlog不支持的类型：%s %+v", column.Name /*col.Name*/, reflect.TypeOf(edata))
			*buf = append(*buf, "\"--unkonw type--\""...)
		} else {
			*buf = append(*buf, "null"...)
		}
	}
}

func (h *binlogHandler) OnRow(e *canal.RowsEvent) error {
	// 发生变化的数据表e.Table，如xsl.x_reports
	// 发生的操作类型e.Action，如update、insert、delete
	// 如update的数据，update的数据以双数出现前面为更新前的数据，后面的为更新后的数据
	// 0，2，4偶数的为更新前的数据，奇数的为更新后的数据
	// [[1 1 3074961 [115 102 103 98 114]   1 1485739538 1485739538]
	// [1 1 3074961 [115 102 103 98 114] 1 1 1485739538 1485739538]]
	// delete一次返回一条数据
	// delete的数据delete [[3 1 3074961 [97 115 100 99 97 100 115] 1,2,2 1 1485768268 1485768268]]
	// 一次插入多条的时候，同时返回
	// insert的数据insert xsl.x_reports [[6 0 0 [] 0 1 0 0]]
	log.Debugf("binlog基础数据：%+v", e.Rows)
	columns_len := len(e.Table.Columns)
	log.Debugf("binlog缓冲区详细信息: %d %d", len(h.buf), cap(h.buf))
	db := []byte(e.Table.Schema)
	point := []byte(".")
	table := []byte(e.Table.Name)
	dblen := len(db) + len(table) + len(point)
	if e.Action == "update" {
		for i := 0; i < len(e.Rows); i += 2 {
			atomic.AddInt64(&h.EventIndex, int64(1))
			buf := h.buf[:0]
			buf = append(buf, byte(dblen))
			buf = append(buf, byte(dblen>>8))
			buf = append(buf, db...)
			buf = append(buf, point...)
			buf = append(buf, table...)
			buf = append(buf, "{\"database\":\""...)
			buf = append(buf, e.Table.Schema...)
			buf = append(buf, "\",\"event\":{\"data\":{\"old_data\":{"...)
			rows_len := len(e.Rows[i])
			for k, col := range e.Table.Columns {
				buf = append(buf, "\""...)
				buf = append(buf, col.Name...)
				buf = append(buf, "\":"...)
				var edata interface{}
				if k < rows_len {
					edata = e.Rows[i][k]
				} else {
					log.Warn("binlog未知的行", col.Name)
					edata = nil
				}
				h.append(&buf, edata, &col)
				if k < columns_len-1 {
					buf = append(buf, ","...)
				}
			}
			buf = append(buf, "},\"new_data\":{"...)
			rows_len = len(e.Rows[i+1])
			for k, col := range e.Table.Columns {
				buf = append(buf, "\""...)
				buf = append(buf, col.Name...)
				buf = append(buf, "\":"...)
				var edata interface{}
				if k < rows_len {
					edata = e.Rows[i+1][k]
				} else {
					log.Warn("binlog未知的行", col.Name)
					edata = nil
				}
				h.append(&buf, edata, &col)
				if k < columns_len-1 {
					buf = append(buf, ","...)
				}
			}
			buf = append(buf, "}},\"event_type\":\""...)
			buf = append(buf, e.Action...)
			buf = append(buf, "\",\"time\":"...)
			buf = strconv.AppendInt(buf, time.Now().Unix(), 10)
			buf = append(buf, "},\"event_index\":"...)
			buf = strconv.AppendInt(buf, h.EventIndex, 10)
			buf = append(buf, ",\"table\":\""...)
			buf = append(buf, e.Table.Name...)
			buf = append(buf, "\"}"...)
			h.notify(buf)
		}
	} else {
		for i := 0; i < len(e.Rows); i += 1 {
			atomic.AddInt64(&h.EventIndex, int64(1))
			buf := h.buf[:0]
			buf = append(buf, byte(dblen))
			buf = append(buf, byte(dblen>>8))
			buf = append(buf, db...)
			buf = append(buf, point...)
			buf = append(buf, table...)
			buf = append(buf, "{\"database\":\""...)
			buf = append(buf, e.Table.Schema...)
			buf = append(buf, "\",\"event\":{\"data\":{"...)
			rows_len := len(e.Rows[i])
			for k, col := range e.Table.Columns {
				buf = append(buf, "\""...)
				buf = append(buf, col.Name...)
				buf = append(buf, "\":"...)
				var edata interface{}
				if k < rows_len {
					edata = e.Rows[i][k]
				} else {
					log.Warn("binlog未知的行", col.Name)
					edata = nil
				}
				h.append(&buf, edata, &col)
				if k < columns_len-1 {
					buf = append(buf, ","...)
				}
			}
			buf = append(buf, "},\"event_type\":\""...)
			buf = append(buf, e.Action...)
			buf = append(buf, "\",\"time\":"...)
			buf = strconv.AppendInt(buf, time.Now().Unix(), 10)
			buf = append(buf, "},\"event_index\":"...)
			buf = strconv.AppendInt(buf, h.EventIndex, 10)
			buf = append(buf, ",\"table\":\""...)
			buf = append(buf, e.Table.Name...)
			buf = append(buf, "\"}"...)
			h.notify(buf)
		}
	}
	return nil
}

func (h *binlogHandler) String() string {
	return "binlogHandler"
}

func (h *binlogHandler) OnRotate(e *replication.RotateEvent) error {
	log.Debugf("OnRotate event fired, %+v", e)
	return nil
}

func (h *binlogHandler) OnDDL(p mysql.Position, e *replication.QueryEvent) error {
	log.Debugf("OnDDL event fired, %+v, %+v", p, e)
	return nil
}

func (h *binlogHandler) OnXID(p mysql.Position) error {
	log.Debugf("OnXID event fired, %+v.", p)
	return nil
}

func (h *binlogHandler) OnGTID(g mysql.GTIDSet) error {
	log.Debugf("OnGTID event fired, GTID: %+v", g)
	return nil
}

func (h *binlogHandler) OnPosSynced(p mysql.Position, b bool) error {
	//h.lock.Lock()
	//defer h.lock.Unlock()
	log.Debugf("binlog事件：OnPosSynced %+v %b", p, b)
	//data := fmt.Sprintf("%s:%d:%d", p.Name, p.Pos, atomic.LoadInt64(&h.EventIndex))
	h.SaveBinlogPostionCache(p.Name, int64(p.Pos), atomic.LoadInt64(&h.EventIndex))
	//h.Cluster.SendPos(data)
	h.lastBinFile = p.Name
	h.lastPos = p.Pos
	return nil
}

func (h *binlogHandler) SaveBinlogPostionCache(binFile string, pos int64, eventIndex int64) {
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
	log.Debugf("write binlog cache: %s, %d, %d, %+v", binFile, pos, eventIndex, r)
	n, err := h.cacheHandler.WriteAt(r, 0)
	h.binlog.Drive.Write(r)
	log.Debugf("%d , %+v", n, err)
	if err != nil || n <= 0 {
		log.Errorf("binlog写入缓存文件错误：%+v", err)
		return
	}
}

func (h *binlogHandler) getBinlogPositionCache() (string, int64, int64) {
	// read 2 bytes is file data length
	l := make([]byte, 2)
	n, err := h.cacheHandler.Read(l)
	if n <= 0 || err != nil {
		//log.Errorf("read error: %+v", err)
		return "", int64(0), int64(0)
	}
	// dl is file data length
	dl := int64(l[0]) | int64(l[1]) << 8
	data := make([]byte, dl)
	h.cacheHandler.Read(data)
	if len(data) < 18 {
		return "", 0, 0
	}
	pos := int64(data[0]) | int64(data[1]) << 8 | int64(data[2]) << 16 |
			int64(data[3]) << 24 | int64(data[4]) << 32 | int64(data[5]) << 40 |
			int64(data[6]) << 48 | int64(data[7]) << 56
	eventIndex := int64(data[8]) | int64(data[9]) << 8 |
			int64(data[10]) << 16 | int64(data[11]) << 24 |
			int64(data[12]) << 32 | int64(data[13]) << 40 |
			int64(data[14]) << 48 | int64(data[15]) << 56
	return string(data[16:]), pos, eventIndex
}
