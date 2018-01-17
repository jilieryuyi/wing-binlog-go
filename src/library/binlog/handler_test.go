package binlog

import (
	"testing"
	"os"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:      true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
	})
	log.SetLevel(5)
}


func TestBinlogHandler_SaveBinlogPostionCache(t *testing.T) {
	binfile := "mysql-bin.000059"
	pos := int64(123456)
	eventIndex := int64(20)

	h := &binlogHandler{}
	var err error
	flag := os.O_RDWR | os.O_CREATE | os.O_SYNC // | os.O_TRUNC
	h.cacheHandler, err = os.OpenFile("/tmp/cache_test.pos", flag, 0755)
	if err != nil {
		t.Errorf("binlog open cache file error: %+v", err)
	}
	h.SaveBinlogPostionCache(binfile, pos, eventIndex)

	s, p, e := h.getBinlogPositionCache()
	log.Debugf("%s, %d, %d", s, p, e)

	log.Debugf("%+v, %+v", []byte(s), []byte(binfile))

	if s != binfile {
		t.Errorf("getBinlogPositionCache binfile error")
	}

	if pos != p {
		t.Errorf("getBinlogPositionCache pos error")
	}

	if e != eventIndex {
		t.Errorf("getBinlogPositionCache eventIndex error")
	}
}
