package path

import (
	"testing"
	"os"
	"fmt"
)

func TestExists(t *testing.T) {
	fmt.Println("path test Exists====================================")
	if !Exists("/usr") {
		t.Error("path check exists error")
	}
	if Exists("/usr/9999999999999999999") {
		t.Error("path check exists error - 2")
	}
}

func TestGetCurrentPath(t *testing.T) {
	fmt.Println("path test GetCurrentPath====================================")
	fmt.Println("current file path: ", os.Args[0])
	fmt.Println("current path:", GetCurrentPath())
}

func TestGetParentPath(t *testing.T) {
	fmt.Println("path test GetParent====================================")

	p := GetParent("/usr/local/")
	if p != "/usr" {
		t.Error("get parent path error - 1")
	}
	p = GetParent("/usr/local")
	if p != "/usr" {
		t.Error("get parent path error - 2")
	}
	p = GetParent("/usr/local.txt")
	if p != "/usr" {
		t.Error("get parent path error - 3")
	}
}

func TestMkdir(t *testing.T) {
	fmt.Println("path test Mkdir====================================")
	dir := "/tmp/1/2/3/4/5/6"
	Mkdir(dir)
	if !Exists(dir) {
		t.Error("mkdir error")
	}
}

func TestGetPath(t *testing.T) {
	fmt.Println("path test GetPath====================================")

	dir := "/usr/local/"
	if "/usr/local" != GetPath(dir) {
		t.Error("get path error")
	}

	dir = "/usr/local/1.text"
	if "/usr/local/1.text" != GetPath(dir) {
		t.Error("get path error - 2")
	}
}
