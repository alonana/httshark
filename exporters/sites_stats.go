package exporters

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
	"sync"
	"time"
)

type SitesStats struct {
	totalStats SingleSiteStats
	hostsStats map[string]SingleSiteStats
	mutex      sync.Mutex
	startTime  time.Time
	buckets    []int
	Logger     *logrus.Logger

}

type SizesStats struct {
	counts map[int]int
	min    *int
	max    *int
}

type SingleSiteStats struct {
	totalSize               uint64
	totalTransactions       uint64
	requestsStats           SizesStats
	responsesStats          SizesStats
	transactionsStats       SizesStats
	requestsWithoutResponse int
}

func (s *SitesStats) init() {
	s.buckets = []int{
		1024,
		5 * 1024,
		50 * 1024,
		100 * 1024,
		256 * 1024,
		1024 * 1024,
		5 * 1024 * 1024,
		10 * 1024 * 1024,
		10024 * 1024 * 1024,
	}

	s.startTime = time.Now()
	s.hostsStats = make(map[string]SingleSiteStats)
	tick := time.NewTicker(core.Config.StatsInterval)
	for {
		select {
		case <-tick.C:
			s.print()
		}
	}
}

func (s *SitesStats) Process(harData *har.Har) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	data, err := json.Marshal(harData)
	if err != nil {
		return fmt.Errorf("marshal har failed: %v", err)
	}

	dataLen := len(data)
	data = nil

	s.totalStats.totalSize += uint64(dataLen)
	s.totalStats.totalTransactions += uint64(len(harData.Log.Entries))
	s.updateSizesStats(&s.totalStats, harData)

	if core.Config.SplitByAppId {
		appId := harData.Log.Entries[0].GetAppId()
		appIdStats := s.hostsStats[appId]
		appIdStats.totalSize += uint64(dataLen)
		appIdStats.totalTransactions += uint64(len(harData.Log.Entries))
		s.updateSizesStats(&appIdStats, harData)
		s.hostsStats[appId] = appIdStats
	}

	return nil
}

func (s *SitesStats) print() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var hosts []string
	for host := range s.hostsStats {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	var messages []string
	titles := "Site,RunSeconds,TotalHarBytes,TotalTransactions,BPS,TPS,AverageHarTransactionBytes,RequestsWithoutResponse"
	titles += s.getSizesTitles("request")
	titles += s.getSizesTitles("response")
	titles += s.getSizesTitles("transaction")
	messages = append(messages, titles)

	message := s.printSingle("__Summary__", s.totalStats)
	messages = append(messages, message)

	for i := 0; i < len(hosts); i++ {
		host := hosts[i]
		message := s.printSingle(host, s.hostsStats[host])
		messages = append(messages, message)
	}

	err := core.SaveToFile(core.Config.SitesStatsFile, strings.Join(messages, "\n"))
	if err != nil {
		s.Logger.Warn("create statistics file failed: %v", err)
		return
	}
}

func (s *SitesStats) getSizesTitles(prefix string) string {
	titles := fmt.Sprintf(",%vMinSize,%vMaxSize", prefix, prefix)
	for i := 0; i < len(s.buckets); i++ {
		bucketSize := s.buckets[i]
		titles += fmt.Sprintf(",%vSizeUpTo%vK", prefix, bucketSize/1024)
	}
	return titles
}

func (s *SitesStats) printSingle(name string, stats SingleSiteStats) string {
	runSeconds := uint64(time.Now().Sub(s.startTime).Seconds())
	bps := float32(stats.totalSize) / float32(runSeconds)
	tps := float32(stats.totalTransactions) / float32(runSeconds)
	avgSize := float32(stats.totalSize) / float32(stats.totalTransactions)
	return fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v",
		name,
		runSeconds,
		stats.totalSize,
		stats.totalTransactions,
		bps,
		tps,
		avgSize,
		stats.requestsWithoutResponse,
		s.printSizesStats(stats.requestsStats),
		s.printSizesStats(stats.responsesStats),
		s.printSizesStats(stats.transactionsStats),
	)
}

func (s *SitesStats) printSizesStats(stats SizesStats) string {
	var line string
	if stats.min == nil || stats.max == nil {
		line = "NA,NA"
	} else {
		line = fmt.Sprintf("%v,%v", *stats.min, *stats.max)
	}
	for i := 0; i < len(s.buckets); i++ {
		bucketSize := s.buckets[i]
		line += fmt.Sprintf(",%v", stats.counts[bucketSize])
	}
	return line
}

func (s *SitesStats) updateSizesStats(stats *SingleSiteStats, data *har.Har) {
	entries := data.Log.Entries
	for i := 0; i < len(entries); i++ {
		entry := entries[i]

		requestSize := entry.Request.BodySize + entry.Request.HeadersSize
		s.updateSizesStatsSingle(&stats.requestsStats, requestSize)
		if entry.Response.Exists {
			responseSize := entry.Response.BodySize + entry.Response.HeadersSize
			s.updateSizesStatsSingle(&stats.responsesStats, responseSize)
			s.updateSizesStatsSingle(&stats.transactionsStats, requestSize+responseSize)
		} else {
			s.updateSizesStatsSingle(&stats.transactionsStats, requestSize)
			stats.requestsWithoutResponse++
		}
	}
}

func (s *SitesStats) updateSizesStatsSingle(stats *SizesStats, size int) {
	if stats.min == nil || size < *stats.min {
		stats.min = &size
	}
	if stats.max == nil || size > *stats.max {
		stats.max = &size
	}

	if stats.counts == nil {
		stats.counts = make(map[int]int)
	}
	for i := 0; i < len(s.buckets); i++ {
		bucketSize := s.buckets[i]
		if size <= bucketSize {
			stats.counts[bucketSize]++
			return
		}
	}
}
