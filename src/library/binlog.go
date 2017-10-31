package library

import (
	"database/sql"
	_ "fmt"
	"fmt"
)
const (
 	COM_REGISTER_SLAVE        = 21;
)

type Binlog struct {
	Db *sql.DB
}

func (log *Binlog) registerSlave(slave_id int) []byte {
	data := make([]byte, 22)

    //COM_BINLOG_DUMP
	data[4]  = byte(COM_REGISTER_SLAVE);
	data[5] = byte(slave_id)
	data[6] = byte(slave_id >> 8)
	data[7] = byte(slave_id >> 16)
	data[8] = byte(slave_id >> 24)


	data[9]  = byte(0)
	data[10] = byte(0)
	data[11] = byte(0)

	data[12] = byte(0);
	data[13] = byte(0 >> 8);

	data[14] = byte(0)
	data[15] = byte(0 >> 8)
	data[16] = byte(0 >> 16)
	data[17] = byte(0 >> 24)

	data[18] = byte(1)
	data[19] = byte(1 >> 8)
	data[20] = byte(1 >> 16)
	data[21] = byte(1 >> 24)

	return data;
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

func (log *Binlog) Register(slave_id int) {
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

	//data := log.registerSlave(slave_id)
	//log.Db;
/*
data = Packet::registerSlave($slave_server_id);

if (!Net::send(data)) {
return false;
}

$result = Net::readPacket();
Packet::success($result);

//封包
data = Packet::binlogDump($this->binlog_file, $this->last_pos, $slave_server_id);

if (!Net::send(data)) {
return false;
}

//认证
$result = Net::readPacket();
Packet::success($result);
return true;*/
}
