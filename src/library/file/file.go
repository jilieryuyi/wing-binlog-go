package file

import (
	"os"
	"io/ioutil"
	log "github.com/sirupsen/logrus"
	"library/path"
	"time"
	"io"
)


// check file is exists
// if exists return true, else return false
func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); err != nil {
		//log.Errorf("check file exists with error: %+v", err)
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

func ReadAt(filePath string, offset int64, length int64) string {
	if !Exists(filePath) {
		log.Warnf("file %s does not exists", filePath)
		return ""
	}
	handle, err := os.Open(filePath)
	if err != nil {
		log.Errorf("read file %s with error: %v", filePath, err)
		return ""
	}
	defer handle.Close()
	//你也可以 Seek 到一个文件中已知的位置并从这个位置开始进行读取。
	//Seek 设置下一次 Read 或 Write 的偏移量为 offset，它的解释取决于
	//whence：
	// 0 表示相对于文件的起始处，
	// 1 表示相对于当前的偏移
	// 2 表示相对于其结尾处。
	nf1, err := handle.Seek(0, io.SeekEnd)
	if err != nil {
		log.Errorf("%v", err)
		return ""
	}
	if length > nf1 {
		length = nf1
	}
	if offset >= 0 {
		if offset > nf1 {
			offset = nf1
		}
		_, err := handle.Seek(offset, io.SeekStart)
		if err != nil {
			log.Println(err)
		}
	} else {
		if offset < 0-nf1 {
			offset = 0 - nf1
		}
		_, err := handle.Seek(offset, io.SeekEnd)
		if err != nil {
			log.Println(err)
		}
	}
	buf := make([]byte, length)
	bytes, err := handle.Read(buf)
	if err != nil || bytes <= 0 {
		log.Warnf("read error: %v", err)
		return ""
	}
	return string(buf)
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
	return n
}

func Delete(filePath string) bool {
	err := os.Remove(filePath)
	return err == nil
}

func Length(filePath string) int64 {
	if !Exists(filePath) {
		return 0
	}
	handle, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer handle.Close()
	//你也可以 Seek 到一个文件中已知的位置并从这个位置开始进行读取。
	//Seek 设置下一次 Read 或 Write 的偏移量为 offset，它的解释取决于
	//whence：
	// 0 表示相对于文件的起始处，
	// 1 表示相对于当前的偏移
	// 2 表示相对于其结尾处。
	//const (
	//    SEEK_SET int = 0 // seek relative to the origin of the file
	//    SEEK_CUR int = 1 // seek relative to the current offset
	//    SEEK_END int = 2 // seek relative to the end
	//)
	length, _ := handle.Seek(0, io.SeekEnd)
	return length

}


