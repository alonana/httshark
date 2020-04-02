package core

import (
	"fmt"
	"os"
	"time"
)

func logWrite(level string, format string, v ...interface{}) {
	formattedTimestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000")
	updatedFormat := fmt.Sprintf("%v %v %v\n", formattedTimestamp, level, format)
	fmt.Printf(updatedFormat, v...)
}

func Error(format string, v ...interface{}) {
	logWrite("ERROR", format, v...)
	os.Exit(1)
}

func Fatal(format string, v ...interface{}) {
	logWrite("FATAL", format, v...)
	os.Exit(1)
}

func Warn(format string, v ...interface{}) {
	logWrite("WARN", format, v...)
}

func Info(format string, v ...interface{}) {
	logWrite("INFO", format, v...)
}

func V1(format string, v ...interface{}) {
	if Config.Verbose < 1 {
		return
	}
	logWrite("VERB", format, v...)
}

func V2(format string, v ...interface{}) {
	if Config.Verbose < 2 {
		return
	}
	logWrite("VERB", format, v...)
}

func V5(format string, v ...interface{}) {
	if Config.Verbose < 5 {
		return
	}
	logWrite("VERB", format, v...)
}
