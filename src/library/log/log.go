package log

import (
    log "github.com/sirupsen/logrus"
    "library/wtime"
    "fmt"
    "library/file"
    "os"
    syslog "log"
)

var (
    // std is the name of the standard logger in stdlib `log`
    std = log.New()
)

func init() {
    fmt.Println("log init------------")
    log.SetFormatter(&log.TextFormatter{TimestampFormat:"2006-01-02 15:04:05",
        ForceColors:true,
        QuoteEmptyFields:true, FullTimestamp:true})
    ResetOutHandler()
}

var cache_log_daytime string = ""
var cache_file *os.File = nil
func ResetOutHandler() {
    syslog.Println("=======>std.Out==", std.Out)
    daytime  := wtime.GetDayTime2()
    if cache_log_daytime == daytime {
        return
    }
    if cache_file != nil {
        syslog.Println("close debug--------1")
        cache_file.Close()
    }
    year     := wtime.GetYear()
    month    := wtime.GetYearMonth()
    dir      := fmt.Sprintf("%s/logs/%d/%s",file.GetCurrentPath(), year, month)
    logs_dir := &file.WPath{dir}
    syslog.Println("try to create dir ", dir)
    if !logs_dir.Exists() {
        os.MkdirAll(dir, 0755)
    }
    syslog.Println("debug--------1")
    log_file    := fmt.Sprintf(dir+"/stdout-%s.log", daytime)
    syslog.Println("debug--------2")
    handle, err := os.OpenFile(log_file, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
    syslog.Println("debug--------3")
    if err == nil {
        cache_log_daytime = daytime
        log.SetOutput(handle)
        cache_file = handle
        syslog.Println("debug--------4")

    } else {
        syslog.Println("open log file error ",err, log_file)
        cache_log_daytime = ""
    }
}

// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
    ResetOutHandler()
    std.Debug(args...)
}

// Print logs a message at level Info on the standard logger.
func Print(args ...interface{}) {
    ResetOutHandler()
    std.Print(args...)
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
    ResetOutHandler()
    std.Info(args...)
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
    ResetOutHandler()
    std.Warn(args...)
}

// Warning logs a message at level Warn on the standard logger.
func Warning(args ...interface{}) {
    ResetOutHandler()
    std.Warning(args...)
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
    ResetOutHandler()
    std.Error(args...)
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
    ResetOutHandler()
    std.Panic(args...)
}

// Fatal logs a message at level Fatal on the standard logger.
func Fatal(args ...interface{}) {
    ResetOutHandler()
    std.Fatal(args...)
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
    ResetOutHandler()
    std.Debugf(format, args...)
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
    ResetOutHandler()
    std.Printf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
    ResetOutHandler()
    std.Infof(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
    ResetOutHandler()
    std.Warnf(format, args...)
}

// Warningf logs a message at level Warn on the standard logger.
func Warningf(format string, args ...interface{}) {
    ResetOutHandler()
    std.Warningf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
    ResetOutHandler()
    std.Errorf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
    ResetOutHandler()
    std.Panicf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger.
func Fatalf(format string, args ...interface{}) {
    ResetOutHandler()
    std.Fatalf(format, args...)
}

// Debugln logs a message at level Debug on the standard logger.
func Debugln(args ...interface{}) {
    ResetOutHandler()
    std.Debugln(args...)
}

// Println logs a message at level Info on the standard logger.
func Println(args ...interface{}) {
    ResetOutHandler()
    std.Println(args...)
}

// Infoln logs a message at level Info on the standard logger.
func Infoln(args ...interface{}) {
    ResetOutHandler()
    std.Infoln(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func Warnln(args ...interface{}) {
    ResetOutHandler()
    std.Warnln(args...)
}

// Warningln logs a message at level Warn on the standard logger.
func Warningln(args ...interface{}) {
    ResetOutHandler()
    std.Warningln(args...)
}

// Errorln logs a message at level Error on the standard logger.
func Errorln(args ...interface{}) {
    ResetOutHandler()
    std.Errorln(args...)
}

// Panicln logs a message at level Panic on the standard logger.
func Panicln(args ...interface{}) {
    ResetOutHandler()
    std.Panicln(args...)
}

// Fatalln logs a message at level Fatal on the standard logger.
func Fatalln(args ...interface{}) {
    ResetOutHandler()
    std.Fatalln(args...)
}

