package aggregated

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"strings"
	"sync"
	"time"
)

var messages map[string]int
var mutex sync.Mutex

func InitLog() {
	messages = make(map[string]int)
	go func() {
		tick := time.NewTicker(core.Config.AggregatedLogInterval)
		for {
			select {
			case <-tick.C:
				printAggregated()
			}
		}
	}()
}

func Warn(format string, v ...interface{}) {
	mutex.Lock()
	defer mutex.Unlock()

	message := fmt.Sprintf(format, v...)
	messages[message]++
}

func printAggregated() {
	mutex.Lock()
	defer mutex.Unlock()

	if len(messages) == 0 {
		return
	}

	var records []string
	for k, v := range messages {
		record := fmt.Sprintf("%v times: %v", v, k)
		records = append(records, record)
	}

	core.Warn("Aggregated logs:\n%v", strings.Join(records, "\n"))

	messages = make(map[string]int)
}
