package library

import (
	"database/sql"
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
	fmt.Println(log.checksum())
	/*$this->checksum = !!$this->getCheckSum();
//$this->slave_server_id = $slave_server_id;
// checksum
if ($this->checksum){
Mysql::query("set @master_binlog_checksum=@@global.binlog_checksum");
}
//heart_period
$heart = 5;
if ($heart) {
Mysql::query("set @master_heartbeat_period=".($heart*1000000000));
}

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
