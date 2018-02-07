package logrus

import (
	"fmt"
	//"log"
	"strings"
)

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

// Level type
type Level uint32

// Convert the Level to a string. E.g. PanicLevel becomes "panic".
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	}

	return "unknown"
}

// ParseLevel takes a string level and returns the Logrus log level constant.
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "panic":
		return PanicLevel, nil
	case "fatal":
		return FatalLevel, nil
	case "error":
		return ErrorLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	}

	var l Level
	return l, fmt.Errorf("not a valid logrus Level: %q", lvl)
}

// A constant exposing all logging levels
var AllLevels = []Level{
	PanicLevel,
	FatalLevel,
	ErrorLevel,
	WarnLevel,
	InfoLevel,
	DebugLevel,
}

// These are the different logging levels. You can set the logging level to log
// on your instance of logger, obtained with `logrus.New()`.
const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel Level = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

// Won't compile if StdLogger can't be realized by a log.Logger
//var (
//	_ StdLogger = &log.Logger{}
//	_ StdLogger = &Entry{}
//	_ StdLogger = &Logger{}
//)

// StdLogger is what your logrus-enabled library should take, that way
// it'll accept a stdlib logger and a logrus logger. There's no standard
// interface, this is the closest we get, unfortunately.
type StdLogger interface {
	Print(int, ...interface{})
	Printf(int, string, ...interface{})
	Println(int, ...interface{})

	Fatal(int, ...interface{})
	Fatalf(int, string, ...interface{})
	Fatalln(int, ...interface{})

	Panic(int, ...interface{})
	Panicf(int, string, ...interface{})
	Panicln(int, ...interface{})
}

// The FieldLogger interface generalizes the Entry and Logger types
type FieldLogger interface {
	WithField(key string, value interface{}) *Entry
	WithFields(fields Fields) *Entry
	WithError(err error) *Entry

	Debugf(int, format string, args ...interface{})
	Infof(int, format string, args ...interface{})
	Printf(int, format string, args ...interface{})
	Warnf(int, format string, args ...interface{})
	Warningf(int, format string, args ...interface{})
	Errorf(int, format string, args ...interface{})
	Fatalf(int, format string, args ...interface{})
	Panicf(int, format string, args ...interface{})

	Debug(skip int, args ...interface{})
	Info(skip int, args ...interface{})
	Print(skip int, args ...interface{})
	Warn(skip int, args ...interface{})
	Warning(skip int, args ...interface{})
	Error(skip int, args ...interface{})
	Fatal(skip int, args ...interface{})
	Panic(skip int, args ...interface{})

	Debugln(skip int, args ...interface{})
	Infoln(skip int, args ...interface{})
	Println(skip int, args ...interface{})
	Warnln(skip int, args ...interface{})
	Warningln(skip int, args ...interface{})
	Errorln(skip int, args ...interface{})
	Fatalln(skip int, args ...interface{})
	Panicln(skip int, args ...interface{})
}
