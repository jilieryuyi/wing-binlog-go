package binlog

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"library/file"
	"library/services"
	wstring "library/string"

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
	data := fmt.Sprintf("%s:%d:%d", p.Name, p.Pos, atomic.LoadInt64(&h.EventIndex))
	h.SaveBinlogPostionCache(data)
	h.Cluster.SendPos(data)
	h.lastBinFile = p.Name
	h.lastPos = p.Pos
	return nil
}

func (h *binlogHandler) setCacheInfo(data string) {
	h.lock.Lock()
	defer h.lock.Unlock()
	res := strings.Split(data, ":")
	if len(res) < 3 {
		log.Errorf("setCacheInfo data error: %s", data)
		return
	}
	h.lastBinFile = res[0]
	wstr := wstring.WString{res[1]}
	h.lastPos = uint32(wstr.ToInt64())
	wstr = wstring.WString{res[2]}
	h.EventIndex = wstr.ToInt64()
}

func (h *binlogHandler) SaveBinlogPostionCache(data string) {
	log.Debugf("binlog写入缓存：%s", data)
	wdata := []byte(data)
	_, err := h.cacheHandler.WriteAt(wdata, 0)
	//select {
	//case <-(*h.ctx).Done():
	//	log.Debugf("服务退出，等待SaveBinlogPostionCache完成，%s", data)
	//	h.wg.Done()
	//default:
	//}
	if err != nil {
		log.Errorf("binlog写入缓存文件错误：%+v", err)
		return
	}
}

func (h *binlogHandler) getBinlogPositionCache() (string, int64, int64) {
	wfile := file.WFile{file.CurrentPath + "/cache/mysql_binlog_position.pos"}
	str := wfile.ReadAll()
	if str == "" {
		return "", int64(0), int64(0)
	}
	res := strings.Split(str, ":")
	if len(res) < 3 {
		return "", int64(0), int64(0)
	}
	wstr := wstring.WString{res[1]}
	pos := wstr.ToInt64()
	wstr2 := wstring.WString{res[2]}
	index := wstr2.ToInt64()
	return res[0], pos, index
}
