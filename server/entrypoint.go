package server

import (
	"bufio"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/core/aggregated"
	"github.com/alonana/httshark/core/log"
	"github.com/alonana/httshark/exporters"
	"github.com/alonana/httshark/httpdump"
	"github.com/alonana/httshark/tshark"
	"github.com/alonana/httshark/tshark/bulk"
	"github.com/alonana/httshark/tshark/correlator"
	"github.com/alonana/httshark/tshark/line"
	"github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
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

func IAmAlive(duration time.Duration, logger *logrus.Logger) {
	timer := time.NewTicker(duration)
	for {
		select {
		case <-timer.C:
			logger.Warn(fmt.Sprintf("httshark. dcva: %v, pid: %v. Oper Status = OK",core.Config.DCVAName, os.Getpid()))
		}
	}
}

// processHealthMonitor publishes health stats info to AWS CloudWatch every "duration"
func processHealthMonitor(duration time.Duration) {
	for {
		<-time.After(duration)
		var numOfGoroutines = runtime.NumGoroutine()
		//var memStats runtime.MemStats
		//runtime.ReadMemStats(&memStats)
		//core.Info("Number of goroutines: %d",numOfGoroutines)
		//core.Info("Mem stats: %v",memStats)
		core.CloudWatchClient.PutMetric("num_of_goroutines", "Count",  float64(numOfGoroutines), "httshark_health_monitor")
	}
}
func (p *EntryPoint) Run() {
	core.Init()
	logger := log.NewLogger()
	logger.Warn(fmt.Sprintf("Starting. Instance Id: %v, PID: %v",core.Config.InstanceId,os.Getpid()))
	aggregated.InitLog()

	go IAmAlive(core.Config.HealthMonitorInterval,logger)
	go reportDroppedPackets(logger)

    /*
	go func() {
		port := 6060 + core.Config.InstanceId
		hostAndPort := fmt.Sprintf("localhost:%v", port)
		logger.Warn(fmt.Sprintf("HTTP SERVER: %v", http.ListenAndServe(hostAndPort, nil)))
	}()
	http.HandleFunc("/", p.health)
	*/

	p.exporterProcessor = exporters.CreateProcessor(logger)
	p.exporterProcessor.Start()

	if core.Config.Capture == "httpdump" {
		httpdump.RunHttpDump(p.exporterProcessor.Queue)
	} else {
		p.correlatorProcessor = correlator.Processor{Processor: p.exporterProcessor.Queue, Logger: logger}
		p.correlatorProcessor.Start()

		p.bulkProcessor = bulk.Processor{HttpProcessor: p.correlatorProcessor.Queue, Logger: logger}
		p.bulkProcessor.Start()

		p.lineProcessor = line.Processor{BulkProcessor: p.bulkProcessor.Queue, Logger: logger}
		p.lineProcessor.Start()

		t := tshark.CommandLine{
			Processor: p.lineProcessor.Queue,
			Logger: logger,
		}
		err := t.Start()
		if err != nil {
			logger.Fatal(fmt.Sprintf("start command failed: %v", err))
		}
	}
	p.signalsChannel = make(chan os.Signal, 1)
	signal.Notify(p.signalsChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-p.signalsChannel

	logger.Info(fmt.Sprintf("Termination initiated. PID: %v",os.Getpid()))
	if core.Config.Capture == "tshark" {
		p.lineProcessor.Stop()
		p.bulkProcessor.Stop()
		p.correlatorProcessor.Stop()
	}
	p.exporterProcessor.Stop()
	logger.Info(fmt.Sprintf("Terminating complete"))
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

//dumpcapDropPacketReport example:
//Packets received/dropped on interface 'eno2': 7467018/3189 (pcap:3189/dumpcap:0/flushed:0/ps_ifdrop:0) (100.0%)
func getPacketDropStats(dumpcapDropPacketReport string) (received float64, dropped float64) {
	leftIdx := strings.Index(dumpcapDropPacketReport,":") + 1
	rightIdx := strings.Index(dumpcapDropPacketReport,"(")
	temp:= strings.TrimSpace(dumpcapDropPacketReport[leftIdx:rightIdx])
	slashIdx := strings.Index(temp,"/")
	received, _ = strconv.ParseFloat(temp[0:slashIdx], 64)
	dropped, _ = strconv.ParseFloat(temp[slashIdx+1:], 64)
	return received,dropped
}

func reportDroppedPackets(logger *logrus.Logger) {
	if fileExists(core.PacketDropFileName){
		lines,err := readLines(core.PacketDropFileName)
		if err != nil {
			logger.Error("Failed to read dropped packets file")
			return
		}
		lastLine := lines[len(lines)-1]
		pipeIdx := strings.Index(lastLine,"|")
		dumpcapReport := lastLine[pipeIdx:]
		received,dropped := getPacketDropStats(dumpcapReport)
		errCnt := 0
		err = core.CloudWatchClient.PutMetric(fmt.Sprintf("%v_received_packets", core.Config.DCVAName), "Count", received, core.NAMESPACE)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to put packet stats (received) in CW: %v",err))
			errCnt++
		}
		err = core.CloudWatchClient.PutMetric(fmt.Sprintf("%v_dropped_packets",core.Config.DCVAName),"Count",dropped,core.NAMESPACE)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to put packet stats (dropped) in CW: %v",err))
			errCnt++
		}
		if errCnt == 0 {
			logger.Info(fmt.Sprintf("Packet metric stats was sent to CloudWatch. received: %v, dropped: %v",received,dropped))
		}
	}
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}


