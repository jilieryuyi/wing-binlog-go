package library

import (
	"log"
	"os"
)

var (
	__err_log_instance *log.Logger
	__log_file_handle  *os.File
)

func init() {
	log.Println("log handle init")
	current_path := GetCurrentPath()
	path := &WPath{current_path + "/logs"}
	//如果日志目录不存在，则自动创建
	path.Mkdir()
	//打开文件
	__log_file_handle, _ = os.OpenFile(current_path+"/logs/wing.log", os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	__err_log_instance = log.New(__log_file_handle, "", log.Ldate|log.Ltime|log.Lshortfile)
	//handle.Close()
}

func GetLogInstance() *log.Logger {
	return __err_log_instance
}

func FreeLogInstance() {
	__log_file_handle.Close()
}
