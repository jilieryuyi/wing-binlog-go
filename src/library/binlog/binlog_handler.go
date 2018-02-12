package binlog

import (
	"sync/atomic"
	"time"
	"library/services"
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	log "github.com/sirupsen/logrus"
	"os"
	"library/path"
	"io"
	"library/app"
	"github.com/siddontang/go-mysql/schema"
	"reflect"
	"bytes"
)

func (h *Binlog) handlerInit() {
	var err error
	mysqlBinlogCacheFile := app.CachePath + "/mysql_binlog_position.pos"
	path.Mkdir(path.GetParent(mysqlBinlogCacheFile))
	flag := os.O_RDWR | os.O_CREATE | os.O_SYNC
	h.cacheHandler, err = os.OpenFile(mysqlBinlogCacheFile, flag, 0755)
	if err != nil {
		log.Panicf("open cache file with error：%s, %+v", mysqlBinlogCacheFile, err)
	}
	h.status ^= cacheHandlerClosed
	h.status |= cacheHAndlerOpened
	f, p, index := h.getBinlogPositionCache()
	h.EventIndex = index
	h.setHandler()
	currentPos, err := h.handler.GetMasterPos()
	if err != nil {
		log.Panicf("get master pos with error：%+v", err)
	}
	log.Debugf("==================>master pos: %+v<==================", currentPos)
	if f != "" && p > 0 {
		h.Config.BinFile = f
		h.Config.BinPos  = uint32(p)
		if f == currentPos.Name && h.Config.BinPos > currentPos.Pos {
			//pos set error, auto start form current pos
			h.Config.BinPos = currentPos.Pos
			log.Warnf("pos set error, auto start form: %d", h.Config.BinPos)
		}
	} else {
		h.Config.BinFile = currentPos.Name
		h.Config.BinPos  = currentPos.Pos
	}
	h.lastBinFile = h.Config.BinFile
	h.lastPos     = uint32(h.Config.BinPos)
	h.posChan     = make(chan []byte, posChanLen)
	log.Debugf("==================>  last pos: (%+v, %+v)<==================", h.lastBinFile, h.lastPos)
	go h.syncSavePosition()
}

func (h *Binlog) syncSavePosition() {
	h.wg.Add(1)
	defer h.wg.Done()
	for {
		select {
		case r, ok := <- h.posChan:
			if !ok {
				return
			}
			log.Debugf("write binlog pos cache: %+v", r)
			if h.status & cacheHAndlerOpened > 0 {
				n, err := h.cacheHandler.WriteAt(r, 0)
				if err != nil || n <= 0 {
					log.Errorf("write binlog cache file with error: %+v", err)
				}
			} else {
				log.Warnf("handler is closed")
			}
		case <- h.ctx.Ctx.Done():
			if len(h.posChan) <= 0 {
				return
			}
		}
	}
}

func (h *Binlog) setHandler()  {
	h.lock.Lock()
	defer h.lock.Unlock()
	cfg, err := canal.NewConfigWithFile(app.ConfigPath + "/canal.toml")
	if err != nil {
		log.Panicf("new canal config with error：%+v", err)
	}
	handler, err := canal.NewCanal(cfg)
	if err != nil {
		log.Panicf("new canal with error：%+v", err)
	}
	h.handler = handler
	h.handler.SetEventHandler(h)
}

func (h *Binlog) RegisterService(name string, s services.Service) {
	h.services[name] = s
}

func (h *Binlog) notify(data map[string] interface{}) {
	log.Debugf("binlog notify: %+v", data)
	for _, service := range h.services {
		service.SendAll(data)
	}
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
			log.Warnf("binlog不支持的类型：%s %+v", column.Name, reflect.TypeOf(edata))
			return edata
		} else {
			return "null"
		}
	}
}

func (h *Binlog) OnRow(e *canal.RowsEvent) error {
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
	rowData := make(map[string] interface{})
	rowData["database"]   = e.Table.Schema
	rowData["event_type"] = e.Action
	rowData["time"]       = time.Now().Unix()
	rowData["table"]      = e.Table.Name

	data := make(map[string] interface{})
	ed   := make(map[string] interface{})

	if e.Action == "update" {
		for i := 0; i < len(e.Rows); i += 2 {
			atomic.AddInt64(&h.EventIndex, int64(1))
			rowData["event_index"] = h.EventIndex
			oldData := make(map[string] interface{})
			newData := make(map[string] interface{})
			rowsLen := len(e.Rows[i])
			for k, col := range e.Table.Columns {
				if k < rowsLen {
					oldData[col.Name] = fieldDecode(e.Rows[i][k], &col)
				} else {
					log.Warn("unknown line", col.Name)
					oldData[col.Name] = nil
				}
			}
			rowsLen = len(e.Rows[i+1])
			for k, col := range e.Table.Columns {
				if k < rowsLen {
					newData[col.Name] = fieldDecode(e.Rows[i+1][k], &col)
				} else {
					log.Warn("unknown line", col.Name)
					newData[col.Name] = nil
				}
			}
			data["old_data"] = oldData
			data["new_data"] = newData
			ed["data"] = data
			rowData["event"] = ed
			h.notify(rowData)
		}
	} else {
		for i := 0; i < len(e.Rows); i += 1 {
			atomic.AddInt64(&h.EventIndex, int64(1))
			rowData["event_index"] = h.EventIndex
			rowsLen := len(e.Rows[i])
			for k, col := range e.Table.Columns {
				if k < rowsLen {
					data[col.Name] = fieldDecode(e.Rows[i][k], &col)
				} else {
					log.Warn("unknown line", col.Name)
					data[col.Name] = nil
				}
			}
			ed["data"] = data
			rowData["event"] = ed
			h.notify(rowData)
		}
	}
	return nil
}

func (h *Binlog) String() string {
	return "Binlog"
}

func (h *Binlog) OnRotate(e *replication.RotateEvent) error {
	log.Debugf("OnRotate event fired, %+v", e)
	return nil
}

func (h *Binlog) OnDDL(p mysql.Position, e *replication.QueryEvent) error {
	log.Debugf("OnDDL event fired, %+v, %+v", p, e)
	return nil
}

func (h *Binlog) OnXID(p mysql.Position) error {
	log.Debugf("OnXID event fired, %+v.", p)
	return nil
}

func (h *Binlog) OnGTID(g mysql.GTIDSet) error {
	log.Debugf("OnGTID event fired, GTID: %+v", g)
	return nil
}

func (h *Binlog) onPosChange(data []byte) {
	if data == nil {
		return
	}
	if len(data) < 19 {
		return
	}
	log.Debugf("onPosChange")
	file, pos, index := unpackPos(data)
	if file == "" || pos <= 0 {
		log.Warnf("error with: %s, %d", file, pos)
		return
	}
	h.lastBinFile = file
	h.lastPos     = uint32(pos)
	h.EventIndex  = index
	r := packPos(file, pos, index)
	h.SaveBinlogPositionCache(r)
}

func (h *Binlog) OnPosSynced(p mysql.Position, b bool) error {
	log.Debugf("OnPosSynced fired with data: %+v %b", p, b)
	eventIndex := atomic.LoadInt64(&h.EventIndex)
	pos := int64(p.Pos)
	data := packPos(p.Name, pos, eventIndex)
	h.SaveBinlogPositionCache(data)
	h.Write(data)
	h.lastBinFile = p.Name
	h.lastPos = p.Pos
	return nil
}

func (h *Binlog) SaveBinlogPositionCache(r []byte) {
	if len(h.posChan) < cap(h.posChan) {
		h.posChan <- r
	} else {
		log.Warnf("posChan full")
	}
}

func (h *Binlog) getBinlogPositionCache() (string, int64, int64) {
	if h.status & cacheHandlerClosed > 0 {
		log.Warnf("handler is closed")
		return "", 0, 0
	}
	data := make([]byte, bytes.MinRead)
	h.cacheHandler.Seek(0, io.SeekStart)
	n, err := h.cacheHandler.Read(data)
	if n <= 0 || err != nil {
		if err != io.EOF {
			log.Errorf("read pos error: %v", err)
		}
		return "", int64(0), int64(0)
	}
	return unpackPos(data)
}
