package file

import (
	"os"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"library/path"
	"time"
)


// check file is exists
// if exists return true, else return false
func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); err != nil {
		log.Errorf("check file exists with error: %+v")
		if os.IsNotExist(err) {
			// file does not exist
			return false
		} else {
			// other error
			return false
		}
	} else {
		//exist
		return true
	}
}

func Read(filePath string) string {
	if !Exists(filePath) {
		return ""
	}
	dat, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Errorf("read file with error: %+v")
		return ""
	}
	return string(dat)
}

func Write(filePath string, data string, append bool) int {
	dir := path.GetParent(filePath)
	path.Mkdir(dir)
	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	handle, err := os.OpenFile(filePath, flag, 0755)
	if err != nil {
		return 0
	}
	defer handle.Close()
	wdata := []byte(data)
	n, err := handle.Write(wdata)
	if err != nil {
		return 0
	}
	dlen := len(data)
	start := time.Now().Unix()
	for {
		if (time.Now().Unix() - start) > 1 {
			break
		}
		if n >= dlen {
			break
		}
		wdata = wdata[n : dlen-n]
		m, err := handle.Write(wdata)
		if err != nil {
			break
		}
		n += m
	}
	//io.WriteString(handle, data)
	//ioutil.WriteFile(file.FilePath, []byte(data), 0755)
	return n
}

