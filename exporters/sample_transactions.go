package exporters

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"time"
)

type SampleTransactions struct {
	printed  int
	sequence int
}

func (s *SampleTransactions) init() {
	tick := time.NewTicker(core.Config.StatsInterval)
	for {
		select {
		case <-tick.C:
			s.printed = core.Config.SampledTransactionsRate
		}
	}
}

func (s *SampleTransactions) Process(harData *har.Har) error {
	entries := harData.Log.Entries
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if s.isRelevant(entry) {
			if s.printed > 0 {
				s.printed--
				err := s.print(entry)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *SampleTransactions) isRelevant(entry har.Entry) bool {
	size := entry.Request.BodySize + entry.Request.HeadersSize
	if entry.Response.Exists {
		size += entry.Response.BodySize + entry.Response.HeadersSize
	}
	return size > 2*1024*1024
}

func (s *SampleTransactions) print(entry har.Entry) error {
	s.sequence++
	path := fmt.Sprintf("%v/%v", core.Config.SampledTransactionsFolder, s.sequence)
	data := fmt.Sprintf("%+v", entry)
	return core.SaveToFile(path, data)
}
