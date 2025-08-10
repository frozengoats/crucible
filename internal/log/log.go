package log

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var lock sync.Mutex

const (
	SILENT int = 0
	ERROR  int = 1
	INFO   int = 2
	DEBUG  int = 3
)

func levelToString(level int) string {
	switch level {
	case SILENT:
		return "SILENT"
	case ERROR:
		return "ERROR"
	case INFO:
		return "INFO"
	case DEBUG:
		return "DEBUG"
	default:
		return ""
	}
}

var logLevel = INFO

func SetLevel(level int) {
	logLevel = level
}

func Log(level int, context []any, formatString string, args ...any) {
	if level > logLevel {
		return
	}

	lock.Lock()
	defer lock.Unlock()

	contextStr := ""
	if len(context) > 0 {
		ctxArr := []string{}
		for i := 0; i < len(context); i += 2 {
			ctxArr = append(ctxArr, fmt.Sprintf("%v=%v", context[i], context[i+1]))
		}
		contextStr = strings.Join(ctxArr, " ") + " "
	}

	now := time.Now().Format(time.TimeOnly)
	logStr := fmt.Sprintf(formatString, args...)
	if level == ERROR {
		logStr = fmt.Sprintf("\033[31m%s\033[0m", logStr)
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s %5s - %s%s\n", now, levelToString(level), contextStr, logStr)
}

func Error(context []any, formatString string, args ...any) {
	Log(ERROR, context, formatString, args...)
}

func Info(context []any, formatString string, args ...any) {
	Log(INFO, context, formatString, args...)
}

func Debug(context []any, formatString string, args ...any) {
	Log(DEBUG, context, formatString, args...)
}
