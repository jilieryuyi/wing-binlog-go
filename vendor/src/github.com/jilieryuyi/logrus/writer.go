package logrus

import (
	"bufio"
	"io"
	"runtime"
)

func (logger *Logger) Writer() *io.PipeWriter {
	return logger.WriterLevel(InfoLevel)
}

func (logger *Logger) WriterLevel(level Level) *io.PipeWriter {
	return NewEntry(logger).WriterLevel(level)
}

func (entry *Entry) Writer() *io.PipeWriter {
	return entry.WriterLevel(InfoLevel)
}

func (entry *Entry) WriterLevel(level Level) *io.PipeWriter {
	reader, writer := io.Pipe()

	var printFunc func(skip int, args ...interface{})

	switch level {
	case DebugLevel:
		printFunc = entry.Debug
	case InfoLevel:
		printFunc = entry.Info
	case WarnLevel:
		printFunc = entry.Warn
	case ErrorLevel:
		printFunc = entry.Error
	case FatalLevel:
		printFunc = entry.Fatal
	case PanicLevel:
		printFunc = entry.Panic
	default:
		printFunc = entry.Print
	}

	go entry.writerScanner(skip, reader, printFunc)
	runtime.SetFinalizer(writer, writerFinalizer)

	return writer
}

func (entry *Entry) writerScanner(skip int, reader *io.PipeReader, printFunc func(skip int, args ...interface{})) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		printFunc(skip, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		entry.Errorf(skip,"Error while reading from Writer: %s", err)
	}
	reader.Close()
}

func writerFinalizer(writer *io.PipeWriter) {
	writer.Close()
}
