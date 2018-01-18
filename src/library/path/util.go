package path

import (
	"strings"
	"os"
	"path/filepath"
	log "github.com/sirupsen/logrus"
)

// check dir is exists
// if exists return true, else return false
func Exists(dir string) bool {
	dir = strings.Replace(dir, "\\", "/", -1)
	_, err := os.Stat(dir)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// get current path
func GetCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Errorf("%+v", err)
		return ""
	}
	return strings.Replace(dir, "\\", "/", -1)
}

// current path
var CurrentPath = GetCurrentPath()