package file

import (
	"testing"
	"fmt"
	log "github.com/sirupsen/logrus"
	"library/platform"
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

func TestWFile_Write(t *testing.T) {
	fmt.Println("===============file Write test start===============")
	file := "/tmp/wing-binlog-go.test"
	if platform.System(platform.IS_WINDOWS) {
		file = "C:/wing-binlog-go.test"
	}
	data := "123456"
	n := Write(file, data, false)
	if n != len(data) {
		t.Errorf("write file error")
	}
	if !Exists(file){
		t.Errorf("write(Exists) file error")
	}
	if Exists("ertwert22ret") {
		t.Error("file exist check error - 1")
	}
	l := Length(file)
	if l != int64(len(data)) {
		t.Errorf("write(length) file error")
	}
	rdata := Read(file)
	if rdata != data {
		t.Errorf("write(read) file error")
	}

	rdata = ReadAt(file, 3, 3)
	if rdata != data[3:6] {
		t.Errorf("ReadAt error")
	}

	rdata = ReadAt(file, 0, 3)
	if rdata != data[0:3] {
		t.Errorf("ReadAt error")
	}

	rdata = ReadAt(file, -3, 3)
	if rdata != data[3:6] {
		t.Errorf("ReadAt error")
	}

	rdata = ReadAt(file, -3, 2)
	if rdata != data[3:5] {
		t.Errorf("ReadAt error")
	}

	data = "123"
	n = Write(file, data, false)
	if n != len(data) {
		t.Errorf("write file error - 2")
	}
	l = Length(file)
	if l != int64(len(data)) {
		t.Errorf("write(length) file error-2")
	}
	rdata = Read(file)
	if rdata != data {
		t.Errorf("write(read) file error - 2")
	}

	data2 := "456"
	n = Write(file, data2, true)
	if n != len(data2) {
		t.Errorf("write file error - 3")
	}
	l = Length(file)
	if l != int64(len(data) + len(data2)) {
		t.Errorf("write(length) file error-3")
	}
	rdata = Read(file)
	if rdata != data + data2 {
		t.Errorf("write(read) file error - 3")
	}
	s := Delete(file)
	if !s {
		t.Errorf("file delete error")
	}
	if Exists(file){
		t.Errorf("file delete error")
	}
}
