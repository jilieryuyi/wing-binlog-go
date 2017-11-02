package library

import "testing"
import "library/platform"

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
        t.Error("ReadAll error: " + str, len(str))
    }
}

func TestWFile_Read(t *testing.T) {
    file := &WFile{"/tmp/__test.txt"}
    if platform.System(platform.IS_WINDOWS) {
        file = &WFile{"C:\\__test.txt"}
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
    str = file.Read(-2, 1)

    if str != "3" {
        t.Error("Read error - 3 "+str)
    }

    str = file.Read(-3, 1)

    if str != "2" {
        t.Error("Read error - 4")
    }

    str = file.Read(-3, 2)

    if str != "23" {
        t.Error("Read error - 5")
    }
}

func TestWFile_Length(t *testing.T) {
    file := &WFile{"/tmp/__test.txt"}
    if platform.System(platform.IS_WINDOWS) {
        file = &WFile{"C:\\__test.txt"}
    } else {
        file = &WFile{"/tmp/__test.txt"}
    }

    if file.Length() != 4 {
        t.Error("Length error")
    }
}
