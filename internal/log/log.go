package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

var (
	output io.Writer = os.Stderr
	logDir           = "/cognitiveos/logs"
)

func Init(dir string) {
	if dir != "" {
		logDir = dir
	}
	if err := os.MkdirAll(logDir, 0755); err == nil {
		f, err := os.OpenFile(filepath.Join(logDir, "cpm.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			output = io.MultiWriter(os.Stderr, f)
		}
	}
}

func Info(format string, args ...interface{}) {
	fmt.Fprintf(output, "INFO[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func Warn(format string, args ...interface{}) {
	fmt.Fprintf(output, "WARN[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func Error(format string, args ...interface{}) {
	fmt.Fprintf(output, "ERROR[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}
