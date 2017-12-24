package file

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	wstr "library/string"
)

func GetCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}


var CurrentPath = GetCurrentPath()

type WPath struct {
	Dir string
}

func (dir *WPath) GetParent() string {
	dir.Dir = strings.Replace(dir.Dir, "\\", "/", -1)
	str := wstr.WString{dir.Dir}
	last_index := strings.LastIndex(str.Substr(0, len(dir.Dir)-1), "/")
	return str.Substr(0, last_index)
}

func (dir *WPath) GetPath() string {
	dir.Dir = strings.Replace(dir.Dir, "\\", "/", -1)
	if string(dir.Dir[len(dir.Dir)-1]) == "/" {
		str := wstr.WString{dir.Dir}
		return str.Substr(0, len(dir.Dir)-1)
	}
	return dir.Dir
}

func (dir *WPath) Exists() bool {
	dir.Dir = strings.Replace(dir.Dir, "\\", "/", -1)
	_, err := os.Stat(dir.Dir)
	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}

	return false
}

func (path *WPath) Mkdir() bool {
	if path.Exists() {
		return true
	}
	err := os.MkdirAll(path.Dir, 0755)

	if err != nil {
		return false
	}
	return true
}
