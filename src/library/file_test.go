package library

import "testing"

func TestWFile_Exists(t *testing.T) {
    file := &WFile{"ertwert22ret"}

    if file.Exists() {
        t.Error("file exist check error - 1")
    }

    file = &WFile{"/bin/sh"}
    if !file.Exists() {
        t.Error("file exist check error - 2")
    }
}
