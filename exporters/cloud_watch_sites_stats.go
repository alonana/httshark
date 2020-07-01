package exporters

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"sync"
	"time"
)

type PeriodicSiteStats struct {
	mutex                   sync.Mutex
	totalSize               uint64
	totalTransactions       uint64
}

func (p *PeriodicSiteStats) reset() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.totalTransactions = 0
	p.totalSize = 0
}
func (p *PeriodicSiteStats) init() {
	if core.Config.SendSiteStatsToCloudWatch {
		tick := time.NewTicker(core.Config.CloudWatchStatsInterval)
		for {
			select {
			case <-tick.C:
				    fmt.Printf("cloud_watch_sites_stats. Number of HTTP exchange: %d, size of HTTP exchange: %d\n", p.totalTransactions,p.totalSize)
					core.CloudWatchClient.PutMetric("total_transactions","Count",
						float64(p.totalTransactions),core.NAMESPACE)
					core.CloudWatchClient.PutMetric("total_size","Bytes",
						float64(p.totalSize),core.NAMESPACE)
					p.reset()
			}
		}
	}
}

func (p *PeriodicSiteStats) Process(harData *har.Har) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	data, err := json.Marshal(harData)
	if err != nil {
		return fmt.Errorf("marshal har failed: %v", err)
	}
	dataLen := len(data)
	data = nil
	p.totalSize += uint64(dataLen)
	p.totalTransactions += uint64(len(harData.Log.Entries))

	return nil
}
