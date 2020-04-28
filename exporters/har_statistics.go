package exporters

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"os"
	"sort"
	"strings"
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

	var messages []string
	messages = append(messages, "Site,RunSeconds,TotalBytes,TotalTransactions,BPS,TPS,AverageTransactionBytes")

	message := s.printSingle("__Summary__", s.totalStats)
	messages = append(messages, message)

	for i := 0; i < len(hosts); i++ {
		host := hosts[i]
		message := s.printSingle(host, s.hostsStats[host])
		messages = append(messages, message)
	}
	s.saveToFile(strings.Join(messages, "\n"))
}

func (s *Stats) printSingle(name string, statsBase StatsBase) string {
	runSeconds := uint64(time.Now().Sub(s.startTime).Seconds())
	bps := float32(statsBase.totalSize) / float32(runSeconds)
	tps := float32(statsBase.totalTransactions) / float32(runSeconds)
	avgSize := float32(statsBase.totalSize) / float32(statsBase.totalTransactions)
	return fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v",
		name,
		runSeconds,
		statsBase.totalSize,
		statsBase.totalTransactions,
		bps,
		tps,
		avgSize,
	)
}

func (s *Stats) saveToFile(data string) {
	f, err := os.Create(core.Config.SitesStatisticsFile)
	if err != nil {
		core.Warn("create statistics file failed: %v", err)
		return
	}

	defer func() {
		err = f.Close()
		if err != nil {
			core.Warn("close statistics file failed: %v", err)
		}
	}()

	_, err = f.Write([]byte(data))
	if err != nil {
		core.Warn("write to statistics file failed: %v", err)
		return
	}
}
