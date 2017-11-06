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
	"strconv"
)

type Binlog struct {
	DB_Config Config
	canal.DummyEventHandler
}

func (h *Binlog) OnRow(e *canal.RowsEvent) error {
	log.Printf("%s %v\n", e.Action, e.Rows)
	return nil
}

func (h *Binlog) String() string {
	return "Binlog"
}


func (h *Binlog) Start() {

	user     := string(h.DB_Config["mysql"]["user"].(string))
	password := string(h.DB_Config["mysql"]["password"].(string))
	port     := string(h.DB_Config["mysql"]["port"].(string))
	host     := string(h.DB_Config["mysql"]["host"].(string))

	bin_file     := string(h.DB_Config["client"]["bin_file"].(string))
	bin_pos_str  := string(h.DB_Config["client"]["bin_pos"].(string))
	//ignore_table := string(h.DB_Config["client"]["ignore_table"].(string))
	bin_pos, _   := strconv.Atoi(bin_pos_str)

	slave_id_str := string(h.DB_Config["client"]["slave_id"].(string))
	slave_id, _  := strconv.Atoi(slave_id_str)

	//db_name := string(config["mysql"]["db_name"].(string))
	//charset := string(config["mysql"]["charset"].(string))
	//db, err := sql.Open("mysql", user+":"+password+"@tcp("+host+":"+port+")/"+db_name+"?charset="+charset)

	cfg         := canal.NewDefaultConfig()
	cfg.Addr     = fmt.Sprintf("%s:%s", host, port)
	cfg.User     = user
	cfg.Password = password//"123456"
	cfg.Flavor   = "mysql"

	cfg.ReadTimeout        = 90*time.Second//*readTimeout
	cfg.HeartbeatPeriod    = 10*time.Second//*heartbeatPeriod
	cfg.ServerID           = uint32(slave_id)
	cfg.Dump.ExecutionPath = ""//mysqldump" 不支持mysqldump写为空
	cfg.Dump.DiscardErr    = false

	c, err := canal.NewCanal(cfg)
	if err != nil {
		fmt.Printf("create canal err %v", err)
		os.Exit(1)
	}

	//c.AddDumpIgnoreTables(seps[0], seps[1]) 设置忽略的数据库和表

	//if len(*tables) > 0 && len(*tableDB) > 0 {
	//	subs := strings.Split(*tables, ",")
	//	c.AddDumpTables(*tableDB, subs...)
	//} else if len(*dbs) > 0 {
	//	subs := strings.Split(*dbs, ",")
	//	c.AddDumpDatabases(subs...)
	//}

	c.SetEventHandler(h)

	startPos := mysql.Position{
		Name: bin_file,
		Pos:  uint32(bin_pos),
	}

	go func() {
		err = c.RunFrom(startPos)
		if err != nil {
			log.Printf("start canal err %v", err)
		}
	}()
}