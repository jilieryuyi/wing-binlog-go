package file

import (
	"io/ioutil"
	"os"
	"log"
	"time"
)

type WFile struct {
	FilePath string
}

func (file *WFile) Exists() bool {
	if _, err := os.Stat(file.FilePath); err != nil {
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

func (file *WFile) ReadAll() string {
	if !file.Exists() {
		return ""
	}
	dat, err := ioutil.ReadFile(file.FilePath)
	if err != nil {
		return ""
	}

	//log.Println(dat)
	//去掉了尾部的换行符
	dlen := len(dat) - 1
	if dlen >= 0 {
		if dat[dlen] == 10 {
			dat = dat[0:dlen]
		}
	}

	//去掉尾部的控制符 windows
	dlen = len(dat) - 1
	if dlen >= 0 {
		if dat[dlen] == 13 {
			dat = dat[0:dlen]
		}
	}

	//去掉尾部的空格
	/*dlen = len(dat) - 1
	  if dat[dlen] == 32 {
	      dat = dat[0:dlen]
	  }*/

	return string(dat)
}

func (file *WFile) Read(offset int64, length int64) string {

	if !file.Exists() {
		return ""
	}
	handle, err := os.Open(file.FilePath)
	if err != nil {
		return ""
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

	nf1, err := handle.Seek(0, os.SEEK_END)
	if err != nil {
		log.Println(err)
	}
	//log.Printf("=====>new offset: %d => %d" , nf1, offset)

	if length > nf1 {
		length = nf1
	}

	if offset >= 0 {
		if offset > nf1 {
			offset = nf1
		}
		_, err := handle.Seek(offset, os.SEEK_SET)
		if err != nil {
			log.Println(err)
		}
		//log.Printf("new offset: %d => %d" , nf, offset)
	} else {
		if offset < 0-nf1 {
			offset = 0 - nf1
		}
		_, err := handle.Seek(offset, os.SEEK_END)
		if err != nil {
			log.Println(err)
		}
		//log.Printf("new offset: %d => %d" , nf, offset)
	}
	buf := make([]byte, length)
	bytes, err := handle.Read(buf)

	if err != nil || bytes <= 0 {
		return ""
	}

	blen := len(buf) - 1
	if buf[blen] == 10 {
		buf = buf[0:blen]
	}

	//kl := len(buf) - 1
	//for k := kl; k >= 0; k-- {
	//	if buf[k] == 0 {
	//		buf = buf[0:k]
	//	}  else {
	//		break
	//	}
	//}

	return string(buf)
}

func (file *WFile) Length() int64 {

	if !file.Exists() {
		return 0
	}
	handle, err := os.Open(file.FilePath)
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

	length, _ := handle.Seek(0, os.SEEK_END)

	return length

}

func (file *WFile) Delete() bool {
	err := os.Remove(file.FilePath)
	return err == nil
}

func (file *WFile) Write(data string, append bool) int {
	//dir := WPath{file.FilePath}
	//dir = WPath{dir.GetParent()}
	//dir.Mkdir()

	flag := os.O_WRONLY | os.O_CREATE | os.O_SYNC
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}

	handle, err := os.OpenFile(file.FilePath, flag, 0755)
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
