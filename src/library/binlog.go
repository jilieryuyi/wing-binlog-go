package library

import (
	"github.com/siddontang/go-mysql/canal"
	"github.com/siddontang/go-mysql/mysql"
	"log"
	"fmt"
	"os"
	//"os/signal"
	//"syscall"
	//"strings"
	"time"
	//"strconv"
	"strings"
	"sync/atomic"
	//"database/sql"
)

type Binlog struct {
	DB_Config *AppConfig
}

type binlogHandler struct{
	Event_index int64
	canal.DummyEventHandler
}

func (h *binlogHandler) OnRow(e *canal.RowsEvent) error {

	atomic.AddInt64(&h.Event_index, int64(1))
	log.Printf("%s %d %v\n", e.Action, len(e.Rows), e.Rows)

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

	return nil
}

func (h *binlogHandler) String() string {
	return "binlogHandler"
}


func (h *Binlog) Start() {

	//user     := string(h.DB_Config["mysql"]["user"].(string))
	//password := string(h.DB_Config["mysql"]["password"].(string))
	//port     := string(h.DB_Config["mysql"]["port"].(string))
	//host     := string(h.DB_Config["mysql"]["host"].(string))
    //
	//bin_file     := string(h.DB_Config["client"]["bin_file"].(string))
	//bin_pos_str  := string(h.DB_Config["client"]["bin_pos"].(string))
	////ignore_table := string(h.DB_Config["client"]["ignore_table"].(string))
	//bin_pos, _   := strconv.Atoi(bin_pos_str)
    //
	//slave_id_str := string(h.DB_Config["client"]["slave_id"].(string))
	//slave_id, _  := strconv.Atoi(slave_id_str)

	//db_name := string(config["mysql"]["db_name"].(string))
	//charset := string(config["mysql"]["charset"].(string))
	//db, err := sql.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+db_name+"?charset="+charset)

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

	c, err := canal.NewCanal(cfg)
	if err != nil {
		log.Printf("create canal err %v", err)
		os.Exit(1)
	}

	//ingore_tables := strings.Split(AppConfig.Client.Ignore_table, ",")
	log.Println(h.DB_Config.Client.Ignore_tables)
	for _, v := range h.DB_Config.Client.Ignore_tables {
		db_table := strings.Split(v, ".")
		c.AddDumpIgnoreTables(db_table[0], db_table[1])
	}
	//c.AddDumpIgnoreTables(seps[0], seps[1]) 设置忽略的数据库和表

	//if len(*tables) > 0 && len(*tableDB) > 0 {
	//	subs := strings.Split(*tables, ",")
	//	c.AddDumpTables(*tableDB, subs...)
	//} else if len(*dbs) > 0 {
	//	subs := strings.Split(*dbs, ",")
	//	c.AddDumpDatabases(subs...)
	//}

	//db, err := sql.Open("mysql",
	//	h.DB_Config.Mysql.User + ":" +
	//	h.DB_Config.Mysql.Password + "@tcp(" +
	//	h.DB_Config.Mysql.Host + ":" + h.DB_Config.Mysql.Port + ")/" +
	//	h.DB_Config.Mysql.DbName + "?charset=" +
	//	h.DB_Config.Mysql.Charset)
    //
	//if nil != err {
	//	log.Println(err)
	//	os.Exit(1)
	//}

	//defer db.Close()

	c.SetEventHandler(&binlogHandler{
		int64(0),
		canal.DummyEventHandler{}})

	startPos := mysql.Position{
		Name: h.DB_Config.Client.Bin_file,
		Pos:  uint32(h.DB_Config.Client.Bin_pos),
	}

	go func() {
		err = c.RunFrom(startPos)
		if err != nil {
			log.Printf("start canal err %v", err)
		}
	}()
}