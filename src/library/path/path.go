package path

import (
	"log"
	"strings"
	"path/filepath"
	"os"
)

func GetCurrentPath() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func substr(s string, pos, length int) string {
	runes := []rune(s)
	l := pos + length
	if l > len(runes) {
		l = len(runes)
	}
	return string(runes[pos:l])
}

func GetParentPath(dirctory string) string {
	return substr(dirctory, 0, strings.LastIndex(dirctory, "/"))
}
