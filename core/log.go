package core

import (
	"fmt"
	"os"
	"sync"
	"time"
)

const dateFormat = "2006-01-02T15:04:05.000"

var snapshot []string
var mutex sync.Mutex

func logWrite(level string, format string, v ...interface{}) {
	formattedTimestamp := time.Now().UTC().Format(dateFormat)
	updatedFormat := fmt.Sprintf("%v %v %v\n", formattedTimestamp, level, format)
	fmt.Printf(updatedFormat, v...)
}

func logSnapshotAppend(format string, v ...interface{}) {
	formattedTimestamp := time.Now().UTC().Format(dateFormat)
	updatedFormat := fmt.Sprintf("%v %v", formattedTimestamp, format)
	message := fmt.Sprintf(updatedFormat, v...)

	mutex.Lock()
	defer mutex.Unlock()

	snapshot = append(snapshot, message)
	if len(snapshot) > Config.LogSnapshotAmount {
		snapshot = snapshot[len(snapshot)-Config.LogSnapshotAmount:]
	}
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
	if Config.Verbose >= 1 {
		logWrite("VERB", format, v...)
	}
	if Config.LogSnapshotLevel >= 1 {
		logSnapshotAppend(format, v...)
	}
}

func V2(format string, v ...interface{}) {
	if Config.Verbose >= 2 {
		logWrite("VERB", format, v...)
	}
	if Config.LogSnapshotLevel >= 2 {
		logSnapshotAppend(format, v...)
	}
}

func V5(format string, v ...interface{}) {
	if Config.Verbose >= 5 {
		logWrite("VERB", format, v...)
	}
	if Config.LogSnapshotLevel >= 5 {
		logSnapshotAppend(format, v...)
	}
}

func snapshotTimer() {
	if Config.LogSnapshotInterval == 0 {
		return
	}

	tick := time.NewTicker(Config.LogSnapshotInterval)
	for {
		select {
		case <-tick.C:
			printSnapshot()
		}
	}
}

func printSnapshot() {
	mutex.Lock()
	clone := make([]string, len(snapshot))
	mutex.Unlock()

	f, err := os.Create(Config.LogSnapshotFile)
	if err != nil {
		Warn("create snapshot file failed: %v", err)
		return
	}

	defer func() {
		err = f.Close()
		if err != nil {
			Warn("close snapshot file failed: %v", err)
		}
	}()

	for i := 0; i < len(clone); i++ {
		line := clone[i]
		_, err := f.Write([]byte(line))
		if err != nil {
			Warn("write to snapshot file failed: %v", err)
			return
		}
		_, err = f.Write([]byte("\n"))
		if err != nil {
			Warn("write to snapshot file failed: %v", err)
			return
		}
	}
}

func LimitedError(err error) string {
	s := err.Error()
	if len(s) > Config.LimitedErrorLength {
		return s[:Config.LimitedErrorLength]
	}
	return s
}
