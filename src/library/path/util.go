package path

import (
	"strings"
	"os"
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
