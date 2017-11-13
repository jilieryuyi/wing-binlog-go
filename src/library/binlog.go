package library

import (
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	//"github.com/siddontang/go-mysql/schema"

	"sync/atomic"
	"fmt"
	"time"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	//"log"
	//"os/signal"
	//"syscall"
	//"strings"
	//"database/sql"
	//"reflect"
	//"encoding/json"
	"strconv"
	"library/services"

)

func init() {
	//fmt.Println("binlog init")
	log.SetFormatter(&log.TextFormatter{TimestampFormat:"2006-01-02 15:04:05",
		ForceColors:true,
		QuoteEmptyFields:true, FullTimestamp:true})
}

type Binlog struct {
	DB_Config *AppConfig
	handler *canal.Canal
	is_connected bool
	binlog_handler binlogHandler
}

type positionCache struct {
	pos mysql.Position
	index int64
}

const (
	MAX_CHAN_FOR_SAVE_POSITION = 128
    defaultBufSize = 4096
	DEFAULT_FLOAT_PREC = 6
)

type binlogHandler struct{
	Event_index int64
	canal.DummyEventHandler
	chan_save_position chan positionCache//mysql.Position//   = make(chan SEND_BODY, MAX_QUEUE)
	buf     []byte
	tcp_service *services.TcpService
	websocket_service *services.WebSocketService
	http_service *services.HttpService
}

func (h *binlogHandler) notify(msg []byte) {
	log.Println("发送广播：", string(msg))
	h.tcp_service.SendAll(msg)
	h.websocket_service.SendAll(msg)
}

func (h *binlogHandler) OnRow(e *canal.RowsEvent) error {

	//log.Println(e.Table.Schema, e.Table.Name,
	//e.Table.Columns,
	//e.Table.Indexes,
	//e.Table.PKColumns)

	//log.Printf("OnRow ==>%s %d %v\n", e.Action, len(e.Rows), e.Rows)

	//sql := "show columns from "+e.Table

	//log.Println(h.Event_index, e.Table, e)
	//发生变化的数据表e.Table，如xsl.x_reports
	//发生的操作类型e.Action，如update、insert、delete
	//如update的数据，update的数据以双数出现前面为更新前的数据，后面的为更新后的数据
	//0，2，4偶数的为更新前的数据，奇数的为更新后的数据
	//[[1 1 3074961 [115 102 103 98 114]   1 1485739538 1485739538]
	// [1 1 3074961 [115 102 103 98 114] 1 1 1485739538 1485739538]]

	//delete一次返回一条数据
	//delete的数据delete [[3 1 3074961 [97 115 100 99 97 100 115] 1,2,2 1 1485768268 1485768268]]

	//一次插入多条的时候，同时返回
	//insert的数据insert xsl.x_reports [[6 0 0 [] 0 1 0 0]]

	clen := len(e.Table.Columns)

	//{"database":"new_yonglibao_c","event":{"data":{"new_data":{"affirm_money":100,"created":1416887067,"days":0,"id":1,"invest_id":11,"pay_type":1,"payback_at":1416887067,"payout_money":100,"user_id":523},
	// "old_data":{"affirm_money":100,"created":1416887067,"days":0,"id":1,"invest_id":111,"pay_type":1,"payback_at":1416887067,"payout_money":100,"user_id":523}}
	// ,"event_type":"update","time":1510309349},"event_index":253131297,"table":"bw_active_payout"}

	log.Println("bufinfo: ",len(h.buf), cap(h.buf))

	if e.Action == "update" {
		for i := 0; i < len(e.Rows); i+=2 {
			atomic.AddInt64(&h.Event_index, int64(1))
			buf := h.buf[:0]
			buf = append(buf, "{\"database\":\""...)
			buf = append(buf, e.Table.Schema...)
			buf = append(buf, "\",\"event\":{\"data\":{\"old_data\":{"...)


			for k, col := range e.Table.Columns {
				buf = append(buf, "\""...)
				buf = append(buf, col.Name...)
				buf = append(buf, "\":"...)
				edata := e.Rows[i][k]
				//log.Println(reflect.TypeOf(edata))
				switch edata.(type) {
				case string:
					buf = append(buf, "\""...)
					for _, v := range []byte(edata.(string)) {
						if v == 34 {
							buf = append(buf, "\\"...)
						}
						buf = append(buf, v)
					}
					buf = append(buf, "\""...)
				case []uint8:
					buf = append(buf, "\""...)
					buf = append(buf, edata.([]byte)...)
					buf = append(buf, "\""...)
				case int:
					buf = strconv.AppendInt(buf, int64(edata.(int)), 10)
				case int8:
					buf = strconv.AppendInt(buf, int64(edata.(int8)), 10)
				case int64:
					buf = strconv.AppendInt(buf, int64(edata.(int64)), 10)
				case int32:
					buf = strconv.AppendInt(buf, int64(edata.(int32)), 10)
				case uint:
					buf = strconv.AppendUint(buf, uint64(edata.(uint)), 10)
				case float64:
					buf = strconv.AppendFloat(buf, edata.(float64), 'f', DEFAULT_FLOAT_PREC, 32)
				case float32:
					buf = strconv.AppendFloat(buf, float64(edata.(float32)), 'f', DEFAULT_FLOAT_PREC, 32)

				default:
					buf = append(buf, "\"--unkonw type--\""...)
				}

				if k < clen - 1 {
					buf = append(buf, ","...)
				}

			}

			buf = append(buf, "},\"new_data\":{"...)

			for k, col := range e.Table.Columns {
				//res1[col.Name] = e.Rows[i][k]
				buf = append(buf, "\""...)
				buf = append(buf, col.Name...)
				buf = append(buf, "\":"...)
				edata := e.Rows[i+1][k]
				switch edata.(type) {
				case string:
					buf = append(buf, "\""...)
					for _, v := range []byte(edata.(string)) {
						if v == 34 {
							buf = append(buf, "\\"...)
						}
						buf = append(buf, v)
					}
					buf = append(buf, "\""...)
				case []uint8:
					buf = append(buf, "\""...)
					buf = append(buf, edata.([]byte)...)
					buf = append(buf, "\""...)
				case int:
					buf = strconv.AppendInt(buf, int64(edata.(int)), 10)
				case int8:
					buf = strconv.AppendInt(buf, int64(edata.(int8)), 10)
				case int64:
					buf = strconv.AppendInt(buf, int64(edata.(int64)), 10)
				case int32:
					buf = strconv.AppendInt(buf, int64(edata.(int32)), 10)
				case uint:
					buf = strconv.AppendUint(buf, uint64(edata.(uint)), 10)
				case float64:
					buf = strconv.AppendFloat(buf, edata.(float64), 'f', DEFAULT_FLOAT_PREC, 32)
				case float32:
					buf = strconv.AppendFloat(buf, float64(edata.(float32)), 'f', DEFAULT_FLOAT_PREC, 32)

				default:
					buf = append(buf, "\"--unkonw type--\""...)
				}

				if k < clen - 1 {
					buf = append(buf, ","...)
				}

			}

			buf = append(buf, "}},\"event_type\":\""...)
			buf = append(buf, e.Action...)
			buf = append(buf, "\",\"time\":"...)
			buf = strconv.AppendInt(buf, time.Now().Unix(), 10)
			buf = append(buf, "},\"event_index\":"...)
			buf = strconv.AppendInt(buf, h.Event_index, 10)
			buf = append(buf, ",\"table\":\""...)
			buf = append(buf, e.Table.Name...)
			buf = append(buf, "\"}"...)

			//fmt.Println(string(buf))
			h.notify(buf)
		}
	} else {
		for i := 0; i < len(e.Rows); i += 1 {
			atomic.AddInt64(&h.Event_index, int64(1))
			buf := h.buf[:0]
			buf = append(buf, "{\"database\":\""...)
			buf = append(buf, e.Table.Schema...)
			buf = append(buf, "\",\"event\":{\"data\":{"...)

			for k, col := range e.Table.Columns {
				buf = append(buf, "\""...)
				buf = append(buf, col.Name...)
				buf = append(buf, "\":"...)

				edata := e.Rows[i][k]

				//log.Println(reflect.TypeOf(edata))
				switch edata.(type) {
				case string:
					buf = append(buf, "\""...)
					for _, v := range []byte(edata.(string)){
						if v == 34 {
							buf = append(buf, "\\"...)
						}
						buf = append(buf, v)
					}
					buf = append(buf, "\""...)

				case []uint8:

					buf = append(buf, "\""...)
					buf = append(buf, string(edata.([]byte))...)
					buf = append(buf, "\""...)

				case int:
					buf = strconv.AppendInt(buf, int64(edata.(int)), 10)
				case int8:
					buf = strconv.AppendInt(buf, int64(edata.(int8)), 10)

				case int64:
					buf = strconv.AppendInt(buf, int64(edata.(int64)), 10)
				case int32:
					buf = strconv.AppendInt(buf, int64(edata.(int32)), 10)
				case uint:
					buf = strconv.AppendUint(buf, uint64(edata.(uint)), 10)
				case float64:
					buf = strconv.AppendFloat(buf, edata.(float64), 'f', DEFAULT_FLOAT_PREC, 64)
				case float32:
					buf = strconv.AppendFloat(buf, float64(edata.(float32)), 'f', DEFAULT_FLOAT_PREC, 64)

				default:
					buf = append(buf, "\"--unkonw type--\""...)
				}

				if k < clen - 1 {
					buf = append(buf, ","...)
				}

			}

			buf = append(buf, "},\"event_type\":\""...)
			buf = append(buf, e.Action...)
			buf = append(buf, "\",\"time\":"...)
			buf = strconv.AppendInt(buf, time.Now().Unix(), 10)
			buf = append(buf, "},\"event_index\":"...)
			buf = strconv.AppendInt(buf, h.Event_index, 10)
			buf = append(buf, ",\"table\":\""...)
			buf = append(buf, e.Table.Name...)
			buf = append(buf, "\"}"...)

			//fmt.Println(string(buf))
			h.notify(buf)
		}
	}


	return nil
}

func (h *binlogHandler) String() string {
	return "binlogHandler"
}

func (h *binlogHandler) OnRotate(e *replication.RotateEvent) error {
	log.Println("OnRotate")
	return nil
}

func (h *binlogHandler) OnDDL(p mysql.Position, e *replication.QueryEvent) error {
	log.Println("OnDDL")
	return nil
}

func (h *binlogHandler) OnXID(p mysql.Position) error {
	log.Println("OnXID")
	//h.SaveBinlogPostionCache(p)
	return nil
}

func (h *binlogHandler) OnGTID(g mysql.GTIDSet) error {
	log.Println("OnGTID ==>", g)
	return nil
}

func (h *binlogHandler) OnPosSynced(p mysql.Position, b bool) error {
	log.Println("OnPosSynced ==>", p, b)
	h.SaveBinlogPostionCache(p)
	return nil
}

func (h *Binlog) Close() {
	if !h.is_connected  {
		return
	}
	h.handler.Close()
	h.is_connected = false
	close(h.binlog_handler.chan_save_position)
}

func (h *binlogHandler) SaveBinlogPostionCache(p mysql.Position) {
	if len(h.chan_save_position) >= MAX_CHAN_FOR_SAVE_POSITION - 10 {
		for k := 0; k <= MAX_CHAN_FOR_SAVE_POSITION - 10; k++ {
			<-h.chan_save_position //丢弃掉未写入的部分数据，优化性能，这里丢弃的pos并不影响最终的结果
		}
	}
	h.chan_save_position <- positionCache{p, atomic.LoadInt64(&h.Event_index)}
}

func (h *Binlog) GetBinlogPostionCache() (string, int64, int64) {
	wfile := WFile{GetCurrentPath() +"/cache/mysql_binlog_position.pos"}
	str := wfile.ReadAll()

	if str == "" {
		return "", int64(0), int64(0)
	}

	res := strings.Split(str, ":")

	if len(res) < 3 {
		return "", int64(0), int64(0)
	}

	wstr := WString{res[1]}
	pos := wstr.ToInt64()

	wstr2 := WString{res[2]}
	index := wstr2.ToInt64()

	return res[0], pos, index
}

func (h *Binlog) Start(tcp_service *services.TcpService,
websocket_service *services.WebSocketService, http_service *services.HttpService) {

	cfg         := canal.NewDefaultConfig()
	cfg.Addr     = fmt.Sprintf("%s:%d", h.DB_Config.Mysql.Host, h.DB_Config.Mysql.Port)
	cfg.User     = h.DB_Config.Mysql.User
	cfg.Password = h.DB_Config.Mysql.Password//"123456"
	cfg.Flavor   = "mysql"

	cfg.ReadTimeout        = 90*time.Second//*readTimeout
	cfg.HeartbeatPeriod    = 10*time.Second//*heartbeatPeriod
	cfg.ServerID           = uint32(h.DB_Config.Client.Slave_id)
	cfg.Dump.ExecutionPath = ""//mysqldump" 不支持mysqldump写为空
	cfg.Dump.DiscardErr    = false

	var err error
	h.handler, err = canal.NewCanal(cfg)
	if err != nil {
		log.Printf("create canal err %v", err)
		os.Exit(1)
	}

	log.Println(h.DB_Config.Client.Ignore_tables)

	for _, v := range h.DB_Config.Client.Ignore_tables {
		db_table := strings.Split(v, ".")
		h.handler.AddDumpIgnoreTables(db_table[0], db_table[1])
	}

	f,p,index := h.GetBinlogPostionCache()

	h.binlog_handler = binlogHandler{Event_index: index}
	var b [defaultBufSize]byte
	h.binlog_handler.buf = b[:0]

	// 3种服务
	h.binlog_handler.tcp_service       = tcp_service
	h.binlog_handler.websocket_service = websocket_service
	h.binlog_handler.http_service      = http_service

	h.binlog_handler.chan_save_position = make(chan positionCache, MAX_CHAN_FOR_SAVE_POSITION)
	h.handler.SetEventHandler(&h.binlog_handler)
	h.is_connected = true

	bin_file := h.DB_Config.Client.Bin_file
	bin_pos  := h.DB_Config.Client.Bin_pos

	if f != "" {
		bin_file = f
	}

	if p > 0 {
		bin_pos = p
	}

	go func() {
		wfile := WFile{GetCurrentPath() +"/cache/mysql_binlog_position.pos"}
		for {
			select {
			case pos := <-h.binlog_handler.chan_save_position:
				if pos.pos.Name != "" && pos.pos.Pos > 0 {
					wfile.Write(fmt.Sprintf("%s:%d:%d", pos.pos.Name, pos.pos.Pos, pos.index), false)
				}
			}
		}
	}()
	go func() {
		startPos := mysql.Position{
			Name: bin_file,
			Pos:  uint32(bin_pos),
		}
		err = h.handler.RunFrom(startPos)
		if err != nil {
			log.Printf("start canal err %v", err)
		}
	}()
}