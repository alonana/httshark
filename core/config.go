package core

import (
	"encoding/json"
	"github.com/namsral/flag"
	"strings"
	"time"
)

type Configuration = struct {
	SampledTransactionsRate     int
	NetworkStreamChannelSize    int
	ChannelBuffer               int
	Verbose                     int
	LogSnapshotLevel            int
	LogSnapshotAmount           int
	LimitedErrorLength          int
	SplitByHost                 bool
	Hosts                       string
	DropContentTypes            string
	HarProcessors               string
	Capture                     string
	Device                      string
	OutputFolder                string
	LogSnapshotFile             string
	SitesStatsFile              string
	RequestsSizesStatsFile      string
	ResponsesSizesStatsFile     string
	SampledTransactionsFolder   string
	LogSnapshotInterval         time.Duration
	ResponseTimeout             time.Duration
	ResponseCheckInterval       time.Duration
	StatsInterval               time.Duration
	AggregatedLogInterval       time.Duration
	ExportInterval              time.Duration
	NetworkStreamChannelTimeout time.Duration
	FullChannelCheckInterval    time.Duration
	FullChannelTimeout          time.Duration
	HealthTransactionTimeout    time.Duration
}

var Config Configuration

func Init() {
	flag.IntVar(&Config.LimitedErrorLength, "limited-error-length", 15, "truncate long errors to this length")
	flag.IntVar(&Config.SampledTransactionsRate, "sample-transactions-rate", 1, "how many transactions should be sampled in each stats interval")
	flag.IntVar(&Config.ChannelBuffer, "channel-buffer", 1, "channel buffer size")
	flag.IntVar(&Config.Verbose, "verbose", 0, "print verbose information. 0=nothing 5=all")
	flag.IntVar(&Config.LogSnapshotLevel, "log-snapshot-level", 0, "print snapshot of logs from verbosity level. 0=nothing 5=all")
	flag.IntVar(&Config.LogSnapshotAmount, "log-snapshot-amount", 0, "print snapshot of logs messages count")
	flag.IntVar(&Config.NetworkStreamChannelSize, "network-stream-channel-size", 1024, "network stream channel size")
	flag.BoolVar(&Config.SplitByHost, "split-by-host", true, "split output files by the request host")
	flag.StringVar(&Config.OutputFolder, "output-folder", ".", "har files output folder")
	flag.StringVar(&Config.Hosts, "hosts", ":80", "comma separated list of IP:port to sample e.g. 1.1.1.1:80,2.2.2.2:9090. To sample all hosts on port 9090, use :9090")
	flag.StringVar(&Config.DropContentTypes, "drop-content-type", "image,audio,video", "comma separated list of content type whose body should be removed (case insensitive, using include for match)")
	flag.StringVar(&Config.Device, "device", "", "interface to use sniffing for")
	flag.StringVar(&Config.Capture, "capture", "tshark", "capture engine to use, one of tshark,httpdump")
	flag.StringVar(&Config.LogSnapshotFile, "log-snapshot-file", "snapshot.log", "logs snapshot file name")
	flag.StringVar(&Config.SitesStatsFile, "sites-stats-file", "statistics.csv", "sites statistics CSV file")
	flag.StringVar(&Config.RequestsSizesStatsFile, "requests-sizes-stats-file", "requests_sizes.csv", "requests sizes statistics CSV file")
	flag.StringVar(&Config.ResponsesSizesStatsFile, "responses-sizes-stats-file", "responses_sizes.csv", "responses sizes statistics CSV file")
	flag.StringVar(&Config.SampledTransactionsFolder, "sampled-transactions-folder", "sampled", "sampled transactions output folder")
	flag.StringVar(&Config.HarProcessors, "har-processors", "file", "comma separated processors of the har file. use any of file,sites-stats,transactions-sizes,sampled-transactions")
	flag.DurationVar(&Config.ResponseTimeout, "response-timeout", time.Minute, "timeout for waiting for response")
	flag.DurationVar(&Config.ResponseCheckInterval, "response-check-interval", 10*time.Second, "check timed out responses interval")
	flag.DurationVar(&Config.ExportInterval, "export-interval", 10*time.Second, "export HAL to file interval")
	flag.DurationVar(&Config.StatsInterval, "stats-interval", 10*time.Second, "print stats exporter interval")
	flag.DurationVar(&Config.LogSnapshotInterval, "log-snapshot-interval", 0, "print log snapshot interval")
	flag.DurationVar(&Config.NetworkStreamChannelTimeout, "network-stream-channel-timeout", 5*time.Second, "network stream go routine accept new packet timeout")
	flag.DurationVar(&Config.AggregatedLogInterval, "aggregated-log-interval", time.Minute, "print aggregated log messages interval")
	flag.DurationVar(&Config.FullChannelCheckInterval, "full-channel-check-interval", 20*time.Millisecond, "check a full channel interval")
	flag.DurationVar(&Config.FullChannelTimeout, "full-channel-timeout", 5*time.Second, "abandon a full channel after this time")
	flag.DurationVar(&Config.HealthTransactionTimeout, "health-transaction-timeout", 10*time.Second, "return error on health if transaction was not received for this period")

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
		if processor != "file" && processor != "sites-stats" && processor != "transactions-sizes" && processor != "sampled-transactions" {
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
