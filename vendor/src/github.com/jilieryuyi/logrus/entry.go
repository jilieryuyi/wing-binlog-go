package logrus

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"time"
	"strings"
	"runtime"
	"path/filepath"
)

var bufferPool *sync.Pool
var workingDir = "/"

func init() {
	bufferPool = &sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}
	wd, err := os.Getwd()
	if err == nil {
		workingDir = filepath.ToSlash(wd) + "/"
	}
}

// Defines the key when adding errors using WithError.
var ErrorKey = "error"

// An entry is the final or intermediate Logrus logging entry. It contains all
// the fields passed with WithField{,s}. It's finally logged when Debug, Info,
// Warn, Error, Fatal or Panic is called on it. These objects can be reused and
// passed around as much as you wish to avoid field duplication.
type Entry struct {
	Logger *Logger

	// Contains all the fields set by the user.
	Data Fields

	// Time at which the log entry was created
	Time time.Time

	// Level the log entry was logged at: Debug, Info, Warn, Error, Fatal or Panic
	// This field will be set on entry firing and the value will be equal to the one in Logger struct field.
	Level Level

	// Message passed to Debug, Info, Warn, Error, Fatal or Panic
	Message string

	// When formatter is called in entry.log(), an Buffer may be set to entry
	Buffer *bytes.Buffer

	// call line
	Line int

	// call form file full path
	FullPath string

	// call form file short path
	ShortPath string

	// call form func
	FuncName string
}

func NewEntry(logger *Logger) *Entry {
	return &Entry{
		Logger: logger,
		// Default is three fields, give a little extra room
		Data: make(Fields, 5),
	}
}

func (entry *Entry) initCallInfo(skip int) {
	pc, fp, ln, ok := runtime.Caller(skip)
	if !ok {
		fmt.Println("error: error during runtime.Caller")
		return
	}
	entry.Line = ln
	entry.FullPath = fp
	if strings.HasPrefix(fp, workingDir) {
		entry.ShortPath = fp[len(workingDir):]
	} else {
		entry.ShortPath = fp
	}
	entry.FuncName = runtime.FuncForPC(pc).Name()
	if strings.HasPrefix(entry.FuncName, workingDir) {
		entry.FuncName = entry.FuncName[len(workingDir):]
	}
}

// Returns the string representation from the reader and ultimately the
// formatter.
func (entry *Entry) String() (string, error) {
	serialized, err := entry.Logger.Formatter.Format(entry)
	if err != nil {
		return "", err
	}
	str := string(serialized)
	return str, nil
}

// Add an error as single field (using the key defined in ErrorKey) to the Entry.
func (entry *Entry) WithError(err error) *Entry {
	return entry.WithField(ErrorKey, err)
}

// Add a single field to the Entry.
func (entry *Entry) WithField(key string, value interface{}) *Entry {
	return entry.WithFields(Fields{key: value})
}

// Add a map of fields to the Entry.
func (entry *Entry) WithFields(fields Fields) *Entry {
	data := make(Fields, len(entry.Data)+len(fields))
	for k, v := range entry.Data {
		data[k] = v
	}
	for k, v := range fields {
		data[k] = v
	}
	return &Entry{Logger: entry.Logger, Data: data}
}

// This function is not declared with a pointer value because otherwise
// race conditions will occur when using multiple goroutines
func (entry Entry) log(skip int, level Level, msg string) {
	entry.initCallInfo(skip+1)

	var buffer *bytes.Buffer
	entry.Time = time.Now()
	entry.Level = level
	entry.Message = msg

	entry.fireHooks()

	buffer = bufferPool.Get().(*bytes.Buffer)
	buffer.Reset()
	defer bufferPool.Put(buffer)
	entry.Buffer = buffer

	entry.write()

	entry.Buffer = nil

	// To avoid Entry#log() returning a value that only would make sense for
	// panic() to use in Entry#Panic(), we avoid the allocation by checking
	// directly here.
	if level <= PanicLevel {
		panic(&entry)
	}
}

func (entry *Entry) fireHooks() {
	entry.Logger.mu.Lock()
	defer entry.Logger.mu.Unlock()
	err := entry.Logger.Hooks.Fire(entry.Level, entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fire hook: %v\n", err)
	}
}

func (entry *Entry) write() {
	serialized, err := entry.Logger.Formatter.Format(entry)
	entry.Logger.mu.Lock()
	defer entry.Logger.mu.Unlock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to obtain reader, %v\n", err)
	} else {
		_, err = entry.Logger.Out.Write(serialized)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
		}
	}
}

func (entry *Entry) Debug(skip int, args ...interface{}) {
	if entry.Logger.level() >= DebugLevel {
		entry.log(skip+1, DebugLevel, fmt.Sprint(args...))
	}
}

func (entry *Entry) Print(skip int,args ...interface{}) {
	entry.Info(skip+1, args...)
}

func (entry *Entry) Info(skip int,args ...interface{}) {
	if entry.Logger.level() >= InfoLevel {
		entry.log(skip+1, InfoLevel, fmt.Sprint(args...))
	}
}

func (entry *Entry) Warn(skip int,args ...interface{}) {
	if entry.Logger.level() >= WarnLevel {
		entry.log(skip+1, WarnLevel, fmt.Sprint(args...))
	}
}

func (entry *Entry) Warning(skip int,args ...interface{}) {
	entry.Warn(skip+1, args...)
}

func (entry *Entry) Error(skip int,args ...interface{}) {
	if entry.Logger.level() >= ErrorLevel {
		entry.log(skip+1, ErrorLevel, fmt.Sprint(args...))
	}
}

func (entry *Entry) Fatal(skip int,args ...interface{}) {
	if entry.Logger.level() >= FatalLevel {
		entry.log(skip+1, FatalLevel, fmt.Sprint(args...))
	}
	Exit(1)
}

func (entry *Entry) Panic(skip int,args ...interface{}) {
	if entry.Logger.level() >= PanicLevel {
		entry.log(skip+1, PanicLevel, fmt.Sprint(args...))
	}
	panic(fmt.Sprint(args...))
}

// Entry Printf family functions

func (entry *Entry) Debugf(skip int,format string, args ...interface{}) {
	if entry.Logger.level() >= DebugLevel {
		entry.Debug(skip+1, fmt.Sprintf(format, args...))
	}
}

func (entry *Entry) Infof(skip int,format string, args ...interface{}) {
	if entry.Logger.level() >= InfoLevel {
		entry.Info(skip+1, fmt.Sprintf(format, args...))
	}
}

func (entry *Entry) Printf(skip int,format string, args ...interface{}) {
	entry.Infof(skip+1, format, args...)
}

func (entry *Entry) Warnf(skip int,format string, args ...interface{}) {
	if entry.Logger.level() >= WarnLevel {
		entry.Warn(skip+1, fmt.Sprintf(format, args...))
	}
}

func (entry *Entry) Warningf(skip int,format string, args ...interface{}) {
	entry.Warnf(skip+1, format, args...)
}

func (entry *Entry) Errorf(skip int,format string, args ...interface{}) {
	if entry.Logger.level() >= ErrorLevel {
		entry.Error(skip+1, fmt.Sprintf(format, args...))
	}
}

func (entry *Entry) Fatalf(skip int,format string, args ...interface{}) {
	if entry.Logger.level() >= FatalLevel {
		entry.Fatal(skip+1, fmt.Sprintf(format, args...))
	}
	Exit(1)
}

func (entry *Entry) Panicf(skip int,format string, args ...interface{}) {
	if entry.Logger.level() >= PanicLevel {
		entry.Panic(skip+1, fmt.Sprintf(format, args...))
	}
}

// Entry Println family functions

func (entry *Entry) Debugln(skip int,args ...interface{}) {
	if entry.Logger.level() >= DebugLevel {
		entry.Debug(skip+1, entry.sprintlnn(args...))
	}
}

func (entry *Entry) Infoln(skip int,args ...interface{}) {
	if entry.Logger.level() >= InfoLevel {
		entry.Info(skip+1, entry.sprintlnn(args...))
	}
}

func (entry *Entry) Println(skip int,args ...interface{}) {
	entry.Infoln(skip+1, args...)
}

func (entry *Entry) Warnln(skip int,args ...interface{}) {
	if entry.Logger.level() >= WarnLevel {
		entry.Warn(skip+1, entry.sprintlnn(args...))
	}
}

func (entry *Entry) Warningln(skip int,args ...interface{}) {
	entry.Warnln(skip+1, args...)
}

func (entry *Entry) Errorln(skip int,args ...interface{}) {
	if entry.Logger.level() >= ErrorLevel {
		entry.Error(skip+1, entry.sprintlnn(args...))
	}
}

func (entry *Entry) Fatalln(skip int,args ...interface{}) {
	if entry.Logger.level() >= FatalLevel {
		entry.Fatal(skip+1, entry.sprintlnn(args...))
	}
	Exit(1)
}

func (entry *Entry) Panicln(skip int,args ...interface{}) {
	if entry.Logger.level() >= PanicLevel {
		entry.Panic(skip+1, entry.sprintlnn(args...))
	}
}

// Sprintlnn => Sprint no newline. This is to get the behavior of how
// fmt.Sprintln where spaces are always added between operands, regardless of
// their type. Instead of vendoring the Sprintln implementation to spare a
// string allocation, we do the simplest thing.
func (entry *Entry) sprintlnn(args ...interface{}) string {
	msg := fmt.Sprintln(args...)
	return msg[:len(msg)-1]
}
