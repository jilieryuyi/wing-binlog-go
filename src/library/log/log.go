package log

import (
	"fmt"
	log "github.com/jilieryuyi/logrus"
	"library/time"
	"os"
	stime "time"
	"library/path"
	"sync"
)

var logHandler = make(map[string] *os.File)
var logLock = new(sync.Mutex)
func getHandler(level log.Level) (*os.File, error) {
	logLock.Lock()
	defer logLock.Unlock()
	//初始化当前，后天的文件句柄
	year     := time.GetYear()
	month    := time.GetYearMonth()
	t        := stime.Now()
	day      := fmt.Sprintf("%d-%02d-%02d", t.Year(), t.Month(), t.Day())
	dir      := fmt.Sprintf("%s/logs/%d/%s", path.CurrentPath, year, month)
	dfile    := fmt.Sprintf("%s/logs/%d/%s/%s-%s.log", path.CurrentPath, year, month, level.String(), day)
	//logsDir := &file.WPath{Dir:dir}
	if !path.Exists(dir) {
		os.MkdirAll(dir, 0755)
	}
	key := fmt.Sprintf("%s%d", day, level)
	var err error
	_, ok := logHandler[key]
	if !ok {
		//初始化当前，后天的文件句柄
		logHandler[key], err = os.OpenFile(dfile, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
		if err != nil {
			return nil, err
		}
	}
	for _key, v := range logHandler{
		if _key != key {
			delete(logHandler, _key)
			v.Close()
		}
	}
	return logHandler[key], nil
}


type ContextHook struct {}
func (hook ContextHook) Levels() []log.Level {
	return log.AllLevels
}
func (hook ContextHook) Fire(entry *log.Entry) error {
	//todo 写入日志文件
	line, err := entry.String()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read entry, %v", err)
		return err
	}
	//fmt.Println("log hook debug: ",line)
	handler, err := getHandler(entry.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "get log handler error, %v", err)
		return nil
	}
	handler.Write([]byte(line))
	//switch entry.Level {
	//case log.PanicLevel:
	//	return hook.Writer.Crit(line)
	//case log.FatalLevel:
	//	return hook.Writer.Crit(line)
	//case log.ErrorLevel:
	//	return hook.Writer.Err(line)
	//case log.WarnLevel:
	//	return hook.Writer.Warning(line)
	//case log.InfoLevel:
	//	return hook.Writer.Info(line)
	//case log.DebugLevel:
	//	return hook.Writer.Debug(line)
	//default:
	//	return nil
	//}


	//if pc, _file, line, ok := runtime.Caller(8); ok {
	//	funcName := runtime.FuncForPC(pc).Name()
	//	entry.Data["file"] = path.Base(_file)
	//	entry.Data["func"] = path.Base(funcName)
	//	entry.Data["line"] = line
	//}
	//pc := make([]uintptr, 3, 3)
	//cnt := runtime.Callers(6, pc)
	//fmt.Printf("\n\n====%+v====\n\n", cnt)
	//for i := 0; i < cnt; i++ {
	//	fu := runtime.FuncForPC(pc[i] - 1)
	//	_file, line := fu.FileLine(pc[i] - 1)
	//	fmt.Printf("\n\n====%+v====%s, %s, %d\n\n", fu, fu.Name(), _file, line)
	//	fu = runtime.FuncForPC(pc[i] - 2)
	//	_file, line = fu.FileLine(pc[i] - 2)
	//	fmt.Printf("\n\n====%+v====%s, %s, %d\n\n", fu, fu.Name(), _file, line)
	//	fu = runtime.FuncForPC(pc[i] - 3)
	//	_file, line = fu.FileLine(pc[i] - 2)
	//	fmt.Printf("\n\n====%+v====%s, %s, %d\n\n", fu, fu.Name(), _file, line)
	//
	//	name := fu.Name()
	//	//if !strings.Contains(name, "github.com/Sirupsen/logrus") {
	//	_file, line = fu.FileLine(pc[i] - 1)
	//	entry.Data["file"] = path.Base(_file)
	//	entry.Data["func"] = path.Base(name)
	//	entry.Data["line"] = line
	//	break
	//	//}
	//}
	return nil
}