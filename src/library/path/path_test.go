package path

import (
    "testing"
    "strings"
)

func TestGetCurrentPath(t *testing.T) {
    path := GetCurrentPath()

    if path == "" {
        t.Error("获取当前目录错误")
    }
}

func TestGetParent(t *testing.T) {
    current_path := GetCurrentPath()
    path := &WPath{current_path}

    runes := []rune(current_path)
    l := 0 + strings.LastIndex(current_path, "/")
    if l > len(runes) {
        l = len(runes)
    }

    if path.GetParent() != string(runes[0:l]) {
        t.Error("获取父目录错误")
    }

    dir := "/usr/local/"
    path = &WPath{dir}

    if path.GetParent() != "/usr" {
        t.Error("获取父目录错误，" + path.GetParent())
    }
}

func TestGetPath(t *testing.T) {
    dir := "/usr/local/"
    path := &WPath{dir}
    if path.GetPath() != dir {
        t.Error("GetPath error")
    }
}

func TestExists(t *testing.T) {
    dir := "/usr/123567567---/"
    path := &WPath{dir}

    exists := path.Exists()

    if exists {
        t.Error("Exists error -1")
    }

    dir = "/"
    path = &WPath{dir}
    if !path.Exists() {
        t.Error("Exists error -2 " + dir)
    }
}