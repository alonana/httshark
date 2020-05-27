package server

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/core/aggregated"
	"github.com/alonana/httshark/exporters"
	"github.com/alonana/httshark/httpdump"
	"github.com/alonana/httshark/tshark"
	"github.com/alonana/httshark/tshark/bulk"
	"github.com/alonana/httshark/tshark/correlator"
	"github.com/alonana/httshark/tshark/line"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

type EntryPoint struct {
	signalsChannel      chan os.Signal
	correlatorProcessor correlator.Processor
	bulkProcessor       bulk.Processor
	lineProcessor       line.Processor
	exporterProcessor   *exporters.Processor
}

// processHealthMonitor publishes health stats info to AWS CloudWatch every "duration"
func processHealthMonitor(duration time.Duration) {
	for {
		<-time.After(duration)
		var numOfGoroutines = runtime.NumGoroutine()
		//var memStats runtime.MemStats
		//runtime.ReadMemStats(&memStats)
		core.Info("Number of goroutines: %d",numOfGoroutines)
		//core.Info("Mem stats: %v",memStats)
		core.CloudWatchClient.PutMetric("num_of_goroutines", "Count",  float64(numOfGoroutines), "httshark_health_monitor")
	}
}
func (p *EntryPoint) Run() {
	core.Init()
	core.Info("Starting. Instance Id: %v, PID: %v",core.Config.InstanceId,os.Getpid())
	aggregated.InitLog()

	http.HandleFunc("/", p.health)
	if core.Config.ActivateHealthMonitor {
		go processHealthMonitor(core.Config.HealthMonitorInterval)
	}


	go func() {
		port := 6060 + core.Config.InstanceId
		hostAndPort := fmt.Sprintf("localhost:%v", port)
		core.Warn("HTTP SERVER: %v", http.ListenAndServe(hostAndPort, nil))
	}()

	p.exporterProcessor = exporters.CreateProcessor()
	p.exporterProcessor.Start()

	if core.Config.Capture == "httpdump" {
		httpdump.RunHttpDump(p.exporterProcessor.Queue)
	} else {
		p.correlatorProcessor = correlator.Processor{Processor: p.exporterProcessor.Queue}
		p.correlatorProcessor.Start()

		p.bulkProcessor = bulk.Processor{HttpProcessor: p.correlatorProcessor.Queue}
		p.bulkProcessor.Start()

		p.lineProcessor = line.Processor{BulkProcessor: p.bulkProcessor.Queue}
		p.lineProcessor.Start()

		t := tshark.CommandLine{
			Processor: p.lineProcessor.Queue,
		}
		err := t.Start()
		if err != nil {
			core.Fatal("start command failed: %v", err)
		}
	}

	p.signalsChannel = make(chan os.Signal, 1)
	signal.Notify(p.signalsChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-p.signalsChannel

	core.Info("Termination initiated")
	if core.Config.Capture == "tshark" {
		p.lineProcessor.Stop()
		p.bulkProcessor.Stop()
		p.correlatorProcessor.Stop()
	}
	p.exporterProcessor.Stop()
	core.Info("Terminating complete")
}


func (p *EntryPoint) health(w http.ResponseWriter, _ *http.Request) {
	err := p.exporterProcessor.CheckHealth()
	if err == nil {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
		return
	}
	w.WriteHeader(500)
	_, _ = w.Write([]byte(err.Error()))
}


