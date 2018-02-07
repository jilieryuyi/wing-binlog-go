package binlog

import (
	"testing"
	"os"
	"fmt"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.SetLevel(log.Level(5))
}

// test SaveBinlogPostionCache api
func TestBinlogHandler_SaveBinlogPostionCache(t *testing.T) {
	binfile := "mysql-bin.000059"
	pos := int64(123456)
	eventIndex := int64(20)
	h := &Binlog{}
	var err error
	flag := os.O_RDWR | os.O_CREATE | os.O_SYNC
	h.cacheHandler, err = os.OpenFile("/tmp/cache_test.pos", flag, 0755)
	if err != nil {
		t.Errorf("binlog open cache file error: %+v", err)
	}
	h.SaveBinlogPostionCache(binfile, pos, eventIndex)
	s, p, e := h.getBinlogPositionCache()
	fmt.Printf("%v, %v, %v\n", s, p, e)
	if s != binfile {
		t.Errorf("getBinlogPositionCache binfile error")
	}
	if pos != p {
		t.Errorf("getBinlogPositionCache pos error")
	}
	if e != eventIndex {
		t.Errorf("getBinlogPositionCache eventIndex error")
	}
	binfile = "mysql-bin.00005"
	pos = int64(12345)
	eventIndex = int64(2)
	h.SaveBinlogPostionCache(binfile, pos, eventIndex)
	s, p, e = h.getBinlogPositionCache()
	fmt.Printf("%v, %v, %v\n", s, p, e)
	if s != binfile {
		t.Errorf("2=>getBinlogPositionCache binfile error")
	}
	if pos != p {
		t.Errorf("2=>getBinlogPositionCache pos error")
	}
	if e != eventIndex {
		t.Errorf("2=>getBinlogPositionCache eventIndex error")
	}
}
