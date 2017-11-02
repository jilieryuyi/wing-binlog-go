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
        file = &WFile{"/bin/sh"}
    }
    if !file.Exists() {
        t.Error("file exist check error - 2")
    }
}
