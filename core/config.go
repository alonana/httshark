package core

import (
	"encoding/json"
	"github.com/namsral/flag"
	"strings"
	"time"
)

type Configuration = struct {
	NetworkStreamChannelSize    int
	ChannelBuffer               int
	Verbose                     int
	LogSnapshotLevel            int
	LogSnapshotAmount           int
	SplitByHost                 bool
	Hosts                       string
	DropContentTypes            string
	HarProcessors               string
	Capture                     string
	Device                      string
	OutputFolder                string
	LogSnapshotFile             string
	SitesStatisticsFile         string
	LogSnapshotInterval         time.Duration
	ResponseTimeout             time.Duration
	ResponseCheckInterval       time.Duration
	StatsInterval               time.Duration
	ExportInterval              time.Duration
	NetworkStreamChannelTimeout time.Duration
}

var Config Configuration

func Init() {
	flag.IntVar(&Config.ChannelBuffer, "channel-buffer", 1, "channel buffer size")
	flag.IntVar(&Config.Verbose, "verbose", 0, "print verbose information. 0=nothing 5=all")
	flag.IntVar(&Config.LogSnapshotLevel, "log-snapshot-level", 0, "print snapshot of logs from verbosity level. 0=nothing 5=all")
	flag.IntVar(&Config.LogSnapshotAmount, "log-snapshot-amount", 0, "print snapshot of logs messages count")
	flag.IntVar(&Config.NetworkStreamChannelSize, "network-stream-channel-size", 1024, "network stream channel size")
	flag.BoolVar(&Config.SplitByHost, "split-by-host", true, "split output files by the request host")
	flag.StringVar(&Config.OutputFolder, "output-folder", ".", "hal files output folder")
	flag.StringVar(&Config.Hosts, "hosts", ":80", "comma separated list of IP:port to sample e.g. 1.1.1.1:80,2.2.2.2:9090. To sample all hosts on port 9090, use :9090")
	flag.StringVar(&Config.DropContentTypes, "drop-content-type", "image,audio,video", "comma separated list of content type whose body should be removed (case insensitive, using include for match)")
	flag.StringVar(&Config.Device, "device", "", "interface to use sniffing for")
	flag.StringVar(&Config.HarProcessors, "har-processors", "file", "comma separated processors of the har file. use any of file,memory,sites-stats")
	flag.StringVar(&Config.Capture, "capture", "tshark", "capture engine to use, one of tshark,httpdump")
	flag.StringVar(&Config.LogSnapshotFile, "log-snapshot-file", "snapshot.log", "logs snapshot file name")
	flag.StringVar(&Config.SitesStatisticsFile, "sites-stats-file", "statistics.csv", "sites statistics CSV file")
	flag.DurationVar(&Config.ResponseTimeout, "response-timeout", 5*time.Minute, "timeout for waiting for response")
	flag.DurationVar(&Config.ResponseCheckInterval, "response-check-interval", 10*time.Second, "check timed out responses interval")
	flag.DurationVar(&Config.ExportInterval, "export-interval", 10*time.Second, "export HAL to file interval")
	flag.DurationVar(&Config.StatsInterval, "stats-interval", 10*time.Second, "print stats exporter interval")
	flag.DurationVar(&Config.LogSnapshotInterval, "log-snapshot-interval", 0, "print log snapshot interval")
	flag.DurationVar(&Config.NetworkStreamChannelTimeout, "network-stream-channel-timeout", 5*time.Second, "network stream go routine accept new packet timeout")

	flag.Parse()
	marshal, err := json.Marshal(Config)
	if err != nil {
		Fatal("marshal config failed: %v", err)
	}

	V5("V5 mode activated")
	V5("common configuration loaded: %v", string(marshal))

	if Config.Device == "" {
		Fatal("device argument must be supplied")
	}
	processors := strings.Split(Config.HarProcessors, ",")
	for i := 0; i < len(processors); i++ {
		processor := processors[i]
		if processor != "file" && processor != "memory" && processor != "sites-stats" {
			Fatal("invalid har processor specified")
		}
	}
	if Config.Capture != "tshark" && Config.Capture != "httpdump" {
		Fatal("invalid capture specified")
	}
	if Config.Hosts == "" {
		Info("hosts were not supplied, will capture all IPs on port 80")
	}

	go snapshotTimer()
}
