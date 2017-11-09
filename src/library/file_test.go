package library

import "testing"
import (
	"library/platform"
	"log"
)

/**
 * 需要注意的是windows下和linux下，文件末尾的换行符是不同的
 * windows下的是 \r\n，而linux下的是 \n
 */

func TestWFile_Exists(t *testing.T) {
	file := &WFile{"ertwert22ret"}

	if file.Exists() {
		t.Error("file exist check error - 1")
	}

	if platform.System(platform.IS_WINDOWS) {
		file = &WFile{"C:\\__test.txt"}
	} else {
		file = &WFile{"/tmp/__test.txt"}
	}
	if !file.Exists() {
		t.Error("file exist check error - 2")
	}
}

func TestWFile_ReadAll(t *testing.T) {
	file := &WFile{"/tmp/__test.txt"}
	if platform.System(platform.IS_WINDOWS) {
		file = &WFile{"C:\\__test.txt"}
	} else {
		file = &WFile{"/tmp/__test.txt"}
	}

	str := file.ReadAll()
	if str != "123" {
		t.Error("ReadAll error: ==>"+str+"<==", len(str))
	}
}

func TestWFile_Read(t *testing.T) {
	file := &WFile{"/tmp/__test.txt"}
	append_len := 1 //"\n"
	if platform.System(platform.IS_WINDOWS) {
		file = &WFile{"C:\\__test.txt"}
		append_len = 2 //"\r\n"
	} else {
		file = &WFile{"/tmp/__test.txt"}
	}

	str := file.Read(0, 1)

	if str != "1" {
		t.Error("Read error - 1")
	}

	str = file.Read(1, 1)

	if str != "2" {
		t.Error("Read error - 2")
	}

	//因为带换行符 所以 -2 才是3
	right_offset := int64(0 - append_len - 1)
	str = file.Read(right_offset, 1)

	if str != "3" {
		t.Error("Read error - 3 " + str)
	}

	right_offset = int64(0 - append_len - 2)
	str = file.Read(right_offset, 1)

	if str != "2" {
		t.Error("Read error - 4")
	}

	right_offset = int64(0 - append_len - 2)
	str = file.Read(right_offset, 2)

	if str != "23" {
		t.Error("Read error - 5")
	}
}

func TestWFile_Length(t *testing.T) {
	file := &WFile{"/tmp/__test.txt"}
	append_len := 1 //"\n"
	if platform.System(platform.IS_WINDOWS) {
		file = &WFile{"C:\\__test.txt"}
		append_len = 2 //"\r\n"
	} else {
		file = &WFile{"/tmp/__test.txt"}
	}

	if file.Length() != int64(3+append_len) {
		t.Error("Length error")
	}
}

func TestWFile_Write(t *testing.T) {
	file := &WFile{"/tmp/__test123.txt"}
	file.Delete()
	n := file.Write("123", true)
	if n != 3 {
		t.Error("write error - 1")
	}

	n = file.Write("456", true)
	if n != 3 {
		t.Error("write error - 2")
	}

	str := file.Read(0, 6)
	if str != "123456" {
		t.Error("write error - 3")
	}

	n = file.Write("123", false)
	if n != 3 {
		t.Error("write error - 4")
	}

	str = file.Read(0, 6)
	log.Println("==>"+str+"<==", len(str), []byte(str))
	if str != "123" {
		t.Error("write error - 5")
	}

	str = file.Read(-6, 6)
	log.Println("==>"+str+"<==", len(str), []byte(str))
	if str != "123" {
		t.Error("write error - 6")
	}
}
