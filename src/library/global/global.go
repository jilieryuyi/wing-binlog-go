package global
import (
	"library/binlog"
)
var binlog *binlog.Binlog = nil

func SetBinlog(_binlog *binlog.Binlog) {
	binlog = _binlog
}

func GetBinlog() *binlog.Binlog {
	return binlog;
}
