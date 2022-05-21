package imghash

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"
)

const (

	// For unrecoverable errors where you would be unable to continue the current scope of code.
	LogError int = iota

	// For non-critical errors that do not require you to abort/exit from the current scope of code.
	LogWarn

	// For non-error "informational" logging.
	LogInfo

	// For any type of verbose debug specific logging.
	LogDebug
)

type Logger struct {
	out    io.Writer
	level  int
	prefix string
}

func NewLogger() *Logger {
	return &Logger{os.Stderr, 0, "LOG"}
}

func NewLoggerWithPrefix(prefix string) *Logger {
	l := NewLogger()
	l.prefix = prefix
	return l
}

func (l *Logger) SetLogLevel(level int) {
	if level < 0 || level > 3 {
		panic("You must set a log level between 0 and 3") // This is lazy but whatever
	}

	l.level = level
}

func (l *Logger) SetOutput(out io.Writer) {
	l.out = out
}

func (l *Logger) Die(format string, a ...interface{}) {
	l.SetLogLevel(LogError)
	l.Error(format, a...)
	os.Exit(1)
}

func (l *Logger) Error(format string, a ...interface{}) {
	if l.level < LogError { // Should never be possible but it's good to check anyways
		return
	}
	l.Print(format, a...)
}

func (l *Logger) Warn(format string, a ...interface{}) {
	if l.level < LogWarn {
		return
	}
	l.Print(format, a...)
}

func (l *Logger) Log(format string, a ...interface{}) {
	if l.level < LogInfo {
		return
	}
	l.Print(format, a...)
}

func (l *Logger) Debug(format string, a ...interface{}) {
	if l.level < LogDebug {
		return
	}
	l.Print(format, a...)
}

func (l *Logger) Println(s string) {
	l.Print(s + "\n")
}

func (l *Logger) Print(format string, a ...interface{}) {
	now := time.Now()

	pc, file, line, _ := runtime.Caller(2) // Go down a depth of two for file and line info
	files := strings.Split(file, "/")
	fns := strings.Split(runtime.FuncForPC(pc).Name(), ".")

	fmt.Fprintf(
		l.out,
		"%s [%s] %s:%d:%s() %s\n",
		now.Format("2006-01-02 15:04:05"),
		l.prefix,
		files[len(files)-1],       // The file name
		line,                      // Line number
		fns[len(fns)-1],           // Function name
		fmt.Sprintf(format, a...), // Message
	)
}
