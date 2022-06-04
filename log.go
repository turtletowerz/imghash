package imghash

import (
	"log"
	"os"
)

const (
	// For unrecoverable errors where you would be unable to continue the current scope of code.
	LogError int = iota

	// For non-critical errors that do not require you to abort/exit from the current scope of code.
	LogWarn

	// For any type of verbose debug specific logging.
	LogDebug
)

type Logger struct {
	*log.Logger
	level int
}

// Returns a new logger with LogError as the logging level.
func NewLogger() *Logger {
	return NewLoggerWithLevel(0)
}

// Returns a new logger with the specified log level set, panicking if the log level is invalid.
func NewLoggerWithLevel(level int) *Logger {
	l := &Logger{log.New(os.Stdout, "", log.LstdFlags), 0}
	l.SetLogLevel(level)
	return l
}

// Sets the given logging level, panicking if the log level is invalid.
func (l *Logger) SetLogLevel(level int) {
	if level < 0 || level > 3 {
		panic("You must set a log level between 0 and 3") // This is lazy but whatever
	}

	l.level = level
}

func (l *Logger) Error(format string, a ...any) {
	if l.level < LogError { // Should never be possible but it's good to check anyways
		return
	}
	l.Fatalf(format, a...)
}

func (l *Logger) Errorln(a ...any) {
	if l.level < LogError { // Should never be possible but it's good to check anyways
		return
	}
	l.Fatalln(a...)
}

func (l *Logger) Warn(format string, a ...any) {
	if l.level < LogWarn {
		return
	}
	l.Printf(format, a...)
}

func (l *Logger) Debug(format string, a ...any) {
	if l.level < LogDebug {
		return
	}
	l.Printf(format, a...)
}

func (l *Logger) Debugln(a ...any) {
	if l.level < LogDebug {
		return
	}
	l.Println(a...)
}

// func (l *Logger) Print(format string, a ...any) {
// 	now := time.Now()

// 	pc, file, line, _ := runtime.Caller(2) // Go down a depth of two for file and line info
// 	files := strings.Split(file, "/")
// 	fns := strings.Split(runtime.FuncForPC(pc).Name(), ".")

// 	fmt.Fprintf(
// 		l.out,
// 		"%s [%s] %s:%d:%s() %s\n",
// 		now.Format("2006-01-02 15:04:05"),
// 		l.prefix,
// 		files[len(files)-1],       // The file name
// 		line,                      // Line number
// 		fns[len(fns)-1],           // Function name
// 		fmt.Sprintf(format, a...), // Message
// 	)
// }
