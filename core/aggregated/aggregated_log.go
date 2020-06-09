package aggregated

import (
	"fmt"
	"github.com/alonana/httshark/core"
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
				publishToCloudWatch()
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

func publishToCloudWatch() {
	mutex.Lock()
	defer mutex.Unlock()

	if len(messages) == 0 {
		return
	}
	for warningDescription, warningCounter := range messages {
		core.CloudWatchClient.PutMetric(warningDescription,"Count",
			float64(warningCounter),"httshark_warnings")
	}
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

	messages = make(map[string]int)
}
