package library

import "os"

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
