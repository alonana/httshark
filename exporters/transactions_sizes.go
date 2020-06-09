package exporters

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"sort"
	"strings"
	"sync"
	"time"
)

type TransactionsSizes struct {
	requests  map[int]int
	responses map[int]int
	mutex     sync.Mutex
}

func (s *TransactionsSizes) init() {
	s.requests = make(map[int]int)
	s.responses = make(map[int]int)
	tick := time.NewTicker(core.Config.StatsInterval)
	for {
		select {
		case <-tick.C:
			s.print()
		}
	}
}

func (s *TransactionsSizes) Process(harData *har.Har) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	entries := harData.Log.Entries
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		requestSize := (entry.Request.HeadersSize + entry.Request.BodySize) / 1024
		s.requests[requestSize] = s.requests[requestSize] + 1
		if entry.Response.Exists {
			responseSize := (entry.Response.HeadersSize + entry.Response.BodySize) / 1024
			s.responses[responseSize] = s.responses[responseSize] + 1
		}
	}

	return nil
}

func (s *TransactionsSizes) print() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.printSizes(s.requests, core.Config.RequestsSizesStatsFile)
	s.printSizes(s.responses, core.Config.ResponsesSizesStatsFile)
}
func (s *TransactionsSizes) printSizes(entities map[int]int, path string) {
	var sizes []int
	for size := range entities {
		sizes = append(sizes, size)
	}
	sort.Ints(sizes)

	var messages []string
	messages = append(messages, "Size,Count")

	for i := 0; i < len(sizes); i++ {
		size := sizes[i]
		message := fmt.Sprintf("%v,%v", size, entities[size])
		messages = append(messages, message)
	}
	err := core.SaveToFile(path, strings.Join(messages, "\n"))
	if err != nil {
		//core.Warn("save sizes statistics file failed: %v", err)
		return
	}
}
