package library

import (
    "os"
    "io/ioutil"
)

type WFile struct{
    File_path string
}

func (file *WFile) Exists() bool {
    if _, err := os.Stat(file.File_path); err != nil {
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
    if (!file.Exists()) {
        return ""
    }
    dat, err := ioutil.ReadFile(file.File_path)
    if err != nil {
        return ""
    }
    return string(dat);
}

func (file *WFile) Read(offset int64, length int64) string {

    if !file.Exists() {
        return ""
    }
    handle, err := os.Open(file.File_path)
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

    handle.Seek(offset, os.SEEK_SET)
    buf := make([]byte, length)
    bytes, err := handle.Read(buf)

    if err != nil || bytes <= 0 {
        return ""
    }

    return string(buf)
}