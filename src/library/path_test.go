package library

import (
    "testing"
    "os"
    "library/platform"
)

func TestGetCurrentPath(t *testing.T) {
    path := GetCurrentPath()

    if path == "" {
        t.Error("获取当前目录错误")
    }
}

func TestWPath_GetParent(t *testing.T) {

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

func TestWPath_GetPath(t *testing.T) {
    dir := "\\usr\\local\\"
    path := &WPath{dir}
    if path.GetPath() != "/usr/local" {
        t.Error("GetPath error")
    }
}

func TestWPath_Exists(t *testing.T) {
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

func TestWPath_Mkdir(t *testing.T) {
    dir := "/tmp/_____________________123____"

    //如果是windows平台，尝试在C盘创建一个测试的文件夹，用完就删除掉
    if platform.System(platform.IS_WINDOWS) {
        dir = "C:\\_____________________123____"
    }
    wpath := &WPath{dir}

    wpath.Mkdir()

    if !wpath.Exists() {
        t.Error("mkdir error")
    }

    os.RemoveAll(dir)
}