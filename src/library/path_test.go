package library

import (
    "testing"
)

func TestGetCurrentPath(t *testing.T) {
    path := GetCurrentPath()

    if path == "" {
        t.Error("获取当前目录错误")
    }
}

func TestGetParent(t *testing.T) {

    dir := "/usr/local"
    path := &WPath{dir}

    if path.GetParent() != "/usr" {
        t.Error("获取父目录错误")
    }

    dir = "/usr/local/"
    path = &WPath{dir}

    if path.GetParent() != "/usr" {
        t.Error("获取父目录错误，" + path.GetParent())
    }
}

func TestGetPath(t *testing.T) {
    dir := "\\usr\\local\\"
    path := &WPath{dir}
    if path.GetPath() != "/usr/local" {
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