package library

import (
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"github.com/siddontang/go-mysql/replication"
	"github.com/siddontang/go-mysql/schema"

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
)

func init() {
	fmt.Println("binlog init")
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

const (
	MAX_CHAN_FOR_SAVE_POSITION = 10240
)

type binlogHandler struct{
	Event_index int64
	canal.DummyEventHandler
	chan_save_position chan mysql.Position//   = make(chan SEND_BODY, MAX_QUEUE)
}

func (h *binlogHandler) OnRow(e *canal.RowsEvent) error {

	log.Println(e.Table.Schema, e.Table.Name,
	e.Table.Columns,
	e.Table.Indexes,
	e.Table.PKColumns)

	log.Printf("OnRow ==>%s %d %v\n", e.Action, len(e.Rows), e.Rows)

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

	if e.Action == "update" {
		for i := 0; i < len(e.Rows); i+=2 {
			atomic.AddInt64(&h.Event_index, int64(1))

			res1 := make(map[string] interface{})
			res2 := make(map[string] interface{})
			for k, col := range e.Table.Columns {
				//log.Println(col.Name, "==>", col.Type, e.Rows[i][k])

				if col.Type == schema.TYPE_STRING {
					wstr1 := WString{e.Rows[i][k]}
					wstr2 := WString{e.Rows[i+1][k]}

					// old data
					res1[col.Name] = wstr1.ToString()
					// new data
					res2[col.Name] = wstr2.ToString()
				} else {
				    //old data
					res1[col.Name] = e.Rows[i][k]
					//new data
					res2[col.Name] = e.Rows[i+1][k]
				}
			}

			event := make(map[string] interface{})

			event["event_type"] = e.Action                     //事件类型
			event["time"]       = time.Now().Unix()            //发生事件的时间戳
			event["data"]       = make(map[string] interface{})//事件数据

			event["data"].(map[string] interface{})["old_data"] = res1                   //更新前的数据
			event["data"].(map[string] interface{})["new_data"] = res2                   //更新后的数据

			result := make(map[string] interface{})
			result["database"]    = e.Table.Schema             //发生事件的数据库
			result["table"]       = e.Table.Name               //发生事件的数据表
			result["event"]       = event                      //事件数据
			result["event_index"] = h.Event_index              //事件原子索引，类型为int64

			//result 就是一个完整的update事件数据
			log.Println(result)
		}



	} else {
		for i := 0; i < len(e.Rows); i+=1 {

			res := make(map[string] interface{})
			for k, col := range e.Table.Columns {
				atomic.AddInt64(&h.Event_index, int64(1))

				//log.Println(col.Name, "==>", col.Type, e.Rows[i][k])

				if col.Type == schema.TYPE_STRING {
					wstr := WString{e.Rows[i][k]}
					res[col.Name] = wstr.ToString()
				} else {
					res[col.Name] = e.Rows[i][k]
				}
			}
			event := make(map[string] interface{})

			event["event_type"] = e.Action                     //事件类型
			event["time"]       = time.Now().Unix()            //发生事件的时间戳
			event["data"]       = make(map[string] interface{})//事件数据
			event["data"]       = res                          //删除的数据

			result := make(map[string] interface{})

			result["database"]    = e.Table.Schema             //发生事件的数据库
			result["table"]       = e.Table.Name               //发生事件的数据表
			result["event"]       = event                      //事件数据
			result["event_index"] = h.Event_index              //事件原子索引，类型为int64

			//result 就是一个完整的update事件数据
			log.Println(result)
		}


	}

	return nil
}

func (h *binlogHandler) String() string {
	return "binlogHandler"
}


func (h *binlogHandler) OnRotate(e *replication.RotateEvent) error {
	log.Println("OnRotate ==>", e.Position, string(e.NextLogName))
	return nil
}
func (h *binlogHandler) OnDDL(p mysql.Position, e *replication.QueryEvent) error {
	log.Println("OnDDL ==>",
		p.Name, p.Pos, e)
	return nil
}
func (h *binlogHandler) OnXID(p mysql.Position) error {
	log.Println("OnXID ==>", p)
	return nil
}
func (h *binlogHandler) OnGTID(g mysql.GTIDSet) error {
	log.Println("OnGTID ==>", g)
	return nil
}
func (h *binlogHandler) OnPosSynced(p mysql.Position, b bool) error {
	//在这里保存pos的位置和bin_file
	log.Println("OnPosSynced ==>", p, b)
	h.chan_save_position <- p
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

func (h *Binlog) GetBinlogPostionCache() (string, int64) {
	wfile := WFile{GetCurrentPath() +"/cache/mysql_binlog_position.pos"}
	str := wfile.ReadAll()

	if str == "" {
		return "", int64(0)
	}

	res := strings.Split(str, ":")

	if len(res) < 2 {
		return res[0], int64(0)
	}

	wstr := WString{res[1]}
	pos := wstr.ToInt64()

	return res[0], pos
}

func (h *Binlog) Start() {

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

	h.binlog_handler = binlogHandler{Event_index: int64(0)}
	h.binlog_handler.chan_save_position = make(chan mysql.Position, MAX_CHAN_FOR_SAVE_POSITION)

	h.handler.SetEventHandler(&h.binlog_handler)
	h.is_connected = true
	//这里要启动一个协程去保存postion

	bin_file := h.DB_Config.Client.Bin_file
	bin_pos  := h.DB_Config.Client.Bin_pos

	f,p := h.GetBinlogPostionCache()

	if f != "" {
		bin_file = f
	}

	if p > 0 {
		bin_pos = p
	}

	startPos := mysql.Position{
		Name: bin_file,
		Pos:  uint32(bin_pos),
	}

	go func() {
		wfile := WFile{GetCurrentPath() +"/cache/mysql_binlog_position.pos"}
		for {
			select {
			case pos := <-h.binlog_handler.chan_save_position:
				log.Println(pos)
				wfile.Write(fmt.Sprintf("%s:%d", pos.Name, pos.Pos), false)
			}
		}
	}()
	go func() {
		err = h.handler.RunFrom(startPos)
		if err != nil {
			log.Printf("start canal err %v", err)
		}
	}()
}