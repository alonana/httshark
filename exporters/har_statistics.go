package exporters

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"sort"
	"sync"
	"time"
)

type Stats struct {
	totalStats StatsBase
	hostsStats map[string]StatsBase
	mutex      sync.Mutex
	startTime  time.Time
}

type StatsBase struct {
	totalSize         uint64
	totalTransactions uint64
}

type PrintStats struct {
	runSeconds            uint64
	totalSize             uint64
	totalTransactions     uint64
	bytesPerSecond        float32
	transactionsPerSecond float32
}

func (s *Stats) init() {
	s.startTime = time.Now()
	s.hostsStats = make(map[string]StatsBase)
	tick := time.NewTicker(core.Config.StatsInterval)
	for {
		select {
		case <-tick.C:
			s.print()
		}
	}
}
func (s *Stats) HarStatistics(harData *har.Har) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	data, err := json.Marshal(harData)
	if err != nil {
		return fmt.Errorf("marshal har failed: %v", err)
	}

	s.totalStats.totalSize += uint64(len(data))
	s.totalStats.totalTransactions += uint64(len(harData.Log.Entries))

	if core.Config.SplitByHost {
		host := harData.Log.Entries[0].GetHost()
		hostStats := s.hostsStats[host]
		hostStats.totalSize += uint64(len(data))
		hostStats.totalTransactions += uint64(len(harData.Log.Entries))
		s.hostsStats[host] = hostStats
	}

	return nil
}

func (s *Stats) print() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var hosts []string
	for host := range s.hostsStats {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	for i := 0; i < len(hosts); i++ {
		host := hosts[i]
		s.printSingle(host, s.hostsStats[host])
	}
	s.printSingle("Summary", s.totalStats)
}

func (s *Stats) printSingle(name string, statsBase StatsBase) {
	runSeconds := uint64(time.Now().Sub(s.startTime).Seconds())
	printStats := PrintStats{
		runSeconds:            runSeconds,
		totalSize:             statsBase.totalSize,
		totalTransactions:     statsBase.totalTransactions,
		bytesPerSecond:        float32(statsBase.totalSize) / float32(runSeconds),
		transactionsPerSecond: float32(statsBase.totalTransactions) / float32(runSeconds),
	}
	core.Info("%v statistics: %+v", name, printStats)
}
