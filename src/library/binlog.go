package library

import (
	"database/sql"
	_ "fmt"
	"fmt"
)

type Binlog struct {
	Db *sql.DB
}

func (log *Binlog) checksum() bool {
	sql_str := "SHOW GLOBAL VARIABLES LIKE 'BINLOG_CHECKSUM'"
	rows, err := log.Db.Query(sql_str)
	if (nil != err) {
		return false;
	}
	defer rows.Close();

	columns, err := rows.Columns()
	if err != nil {
		return false
	}

	clen := len(columns)
	scanArgs := make([]interface{}, clen)
	values   := make([]interface{}, clen)

	for i := range values {
		scanArgs[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scanArgs...)
		for i, col := range values {
			if col != nil {
				if columns[i] == "Value" {
					return string(col.([]byte)) != ""
				}
			}
		}
	}

	return false
}

func (log *Binlog) Register(slave_id string) {
	checksum := log.checksum()

	if checksum {
		log.Db.Query("set @master_binlog_checksum=@@global.binlog_checksum");
	}

	//设置心跳
	heart := 5;
	if heart > 0 {
		sql_str := fmt.Sprintf("set @master_heartbeat_period=%d", heart*1000000000)
		log.Db.Query(sql_str);
	}
/*
$data = Packet::registerSlave($slave_server_id);

if (!Net::send($data)) {
return false;
}

$result = Net::readPacket();
Packet::success($result);

//封包
$data = Packet::binlogDump($this->binlog_file, $this->last_pos, $slave_server_id);

if (!Net::send($data)) {
return false;
}

//认证
$result = Net::readPacket();
Packet::success($result);
return true;*/
}
