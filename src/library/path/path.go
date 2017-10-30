package path

import (
	"log"
	"strings"
	"path/filepath"
	"os"
	"library"
)

func GetCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}


type WPath struct {
	Dir string
}

func (dir *WPath) GetParent() string {
	str := library.WString{dir.Dir}
	return str.Substr(0, strings.LastIndex(dir.Dir, "/"))
}

func (dir *WPath) GetPath() string {
	return dir.Dir
}

func (dir *WPath) Exists() bool {
	_, err := os.Stat(dir.Dir)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	return false
}


