package core

import (
	"encoding/json"
	"fmt"
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
	DumpCapBufferSize           int
	InstanceId                  int
	S3ExporterMaxNumOfEntries   int
	RotateFileMaxSize           int
	RotateFileMaxBackups        int
	RotateFileMaxAge            int
	SplitByHost                 bool
	SplitByAppId                bool
	ActivateHealthMonitor       bool
	SendSiteStatsToCloudWatch   bool
	S3ExporterShouldCompress    bool
	AWSDisableSSL               bool
	UseCloudWatchLoggerHook     bool
	IgnoreHealthCheck           bool
	Hosts                       string
	KeepContentTypes            string
	HarProcessors               string
	Capture                     string
	Device                      string
	OutputFolder                string
	LogSnapshotFile             string
	SitesStatsFile              string
	RequestsSizesStatsFile      string
	ResponsesSizesStatsFile     string
	SampledTransactionsFolder   string
	AWSRegion                   string
	DCVAName                    string
	S3ExporterBucketName        string
	CloudWatchLogLevels         string
	RotateFileLevel             string
	RotateFileFileName          string
	S3ExporterPurgeInterval     time.Duration
	LogSnapshotInterval         time.Duration
	ResponseTimeout             time.Duration
	ResponseCheckInterval       time.Duration
	StatsInterval               time.Duration
	CloudWatchStatsInterval     time.Duration
	HealthMonitorInterval       time.Duration
	AggregatedLogInterval       time.Duration
	ExportInterval              time.Duration
	NetworkStreamChannelTimeout time.Duration
	FullChannelCheckInterval    time.Duration
	FullChannelTimeout          time.Duration
	HealthTransactionTimeout    time.Duration

}

var Config Configuration
var supportedProcessors = map[string]bool{
	"file": true,
	"sites-stats":   true,
	"cw-sites-stats": true,
	"transactions-sizes": true,
	"sampled-transactions": true,
	"s3": true,
}
var args = make([]string,1)

func grabFlagProperties(f *flag.Flag) {
	entry := fmt.Sprintf("{'name':'%s', 'val':'%s','def_val':'%s','usage':'%s'}",f.Name,f.Value,f.DefValue,f.Usage)
	args = append(args, entry)
}
func Init() {
	exporters := make([]string, 0, len(supportedProcessors))
	for k := range supportedProcessors {
		exporters = append(exporters, k)
	}
	exportersStr := strings.Join(exporters,",")
	// see https://godoc.org/gopkg.in/natefinch/lumberjack.v2
	flag.IntVar(&Config.RotateFileMaxSize, "rotate-file-max-size", 50, "max size of rotated file (MB)")
	flag.IntVar(&Config.RotateFileMaxBackups, "rotate-file-max-backups", 20, "max number of rotated file backups")
	flag.IntVar(&Config.RotateFileMaxAge, "rotate-file-max-age", 100, "max number of days to keep files")
	flag.IntVar(&Config.LimitedErrorLength, "limited-error-length", 15, "truncate long errors to this length")
	flag.IntVar(&Config.DumpCapBufferSize, "dumpcap-buffer-size", 20, "capture buffer size (in MiB)")
	flag.IntVar(&Config.InstanceId, "instance-id", 0, "when running in a cluster we identify each instance by this id")
	flag.IntVar(&Config.SampledTransactionsRate, "sample-transactions-rate", 1, "how many transactions should be sampled in each stats interval")
	flag.IntVar(&Config.ChannelBuffer, "channel-buffer", 1, "channel buffer size")
	flag.IntVar(&Config.Verbose, "verbose", 0, "print verbose information. 0=nothing 5=all")
	flag.IntVar(&Config.LogSnapshotLevel, "log-snapshot-level", 0, "print snapshot of logs from verbosity level. 0=nothing 5=all")
	flag.IntVar(&Config.LogSnapshotAmount, "log-snapshot-amount", 0, "print snapshot of logs messages count")
	flag.IntVar(&Config.NetworkStreamChannelSize, "network-stream-channel-size", 1024, "network stream channel size")
	flag.IntVar(&Config.S3ExporterMaxNumOfEntries, "s3-exporter-max-num-of-entries-to-hold", 1024, "max number of entries to accumulate before sending to s3")
	flag.BoolVar(&Config.AWSDisableSSL, "aws-disable-ssl", false, "disable ssl while using AWS API")
	flag.BoolVar(&Config.UseCloudWatchLoggerHook, "use-cw-logger-hook", true, "Use CW logger hook")
	flag.BoolVar(&Config.SplitByHost, "split-by-host", true, "split output files by the request host")
	flag.BoolVar(&Config.SplitByAppId, "split-by-appid", true, "split output files by the app id")
	flag.BoolVar(&Config.ActivateHealthMonitor, "activate-health-monitor", true, "send health stats to AWS CloudWatch")
	flag.BoolVar(&Config.S3ExporterShouldCompress, "s3-exporter-compress", true, "compress the HAR before you dump it to s3")
	flag.BoolVar(&Config.SendSiteStatsToCloudWatch, "send-sites-stats-to-cloudwatch", true, "send site stats stats to AWS CloudWatch")
	flag.BoolVar(&Config.IgnoreHealthCheck, "ignore-hc", true, "do not dump cwaf HC calls")
	flag.StringVar(&Config.OutputFolder, "output-folder", ".", "har files output folder")
	flag.StringVar(&Config.Hosts, "hosts", ":80", "comma separated list of IP:port to sample e.g. 1.1.1.1:80,2.2.2.2:9090. To sample all hosts on port 9090, use :9090")
	flag.StringVar(&Config.KeepContentTypes, "keep-content-type", "json", "comma separated list of content type whose body should be kept (case insensitive, using include for match)")
	flag.StringVar(&Config.Device, "device", "", "interface to use sniffing for")
	flag.StringVar(&Config.Capture, "capture", "tshark", "capture engine to use, one of tshark,httpdump")
	flag.StringVar(&Config.LogSnapshotFile, "log-snapshot-file", "snapshot.log", "logs snapshot file name")
	flag.StringVar(&Config.SitesStatsFile, "sites-stats-file", "statistics.csv", "sites statistics CSV file")
	flag.StringVar(&Config.RequestsSizesStatsFile, "requests-sizes-stats-file", "requests_sizes.csv", "requests sizes statistics CSV file")
	flag.StringVar(&Config.ResponsesSizesStatsFile, "responses-sizes-stats-file", "responses_sizes.csv", "responses sizes statistics CSV file")
	flag.StringVar(&Config.SampledTransactionsFolder, "sampled-transactions-folder", "sampled", "sampled transactions output folder")
	flag.StringVar(&Config.AWSRegion, "aws-region", "us-east-1", "AWS Region")
	flag.StringVar(&Config.DCVAName, "dcva-name", "undefined-dcva", "DCVA name")
	flag.StringVar(&Config.S3ExporterBucketName, "s3-bucket-name", "", "S3 bucket name")
	flag.StringVar(&Config.HarProcessors, "har-processors", "file",exportersStr)
	flag.StringVar(&Config.CloudWatchLogLevels, "cw-log-levels", "panic,fatal,error,warn","a comma delimited string of log levels to be written to cloud watch")
	flag.StringVar(&Config.RotateFileLevel, "rotate-file-min-level", "trace","the min level to write to the file")
	flag.StringVar(&Config.RotateFileFileName, "rotate-file-name", "httshark.log","log file name")
	flag.DurationVar(&Config.ResponseTimeout, "response-timeout", time.Minute, "timeout for waiting for response")
	flag.DurationVar(&Config.S3ExporterPurgeInterval, "s3-exporter-purge-interval", 1*time.Minute, "timeout for exporting data to s3")
	flag.DurationVar(&Config.ResponseCheckInterval, "response-check-interval", 10*time.Second, "check timed out responses interval")
	flag.DurationVar(&Config.HealthMonitorInterval, "health-monitor-interval", 1*time.Minute, "publish I am alive")
	flag.DurationVar(&Config.CloudWatchStatsInterval, "cloud-watch-stats-interval", 1*time.Minute, "publish traffic stats to cloud watch interval")
	flag.DurationVar(&Config.ExportInterval, "export-interval", 10*time.Second, "export HAL to file interval")
	flag.DurationVar(&Config.StatsInterval, "stats-interval", 10*time.Second, "print stats exporter interval")
	flag.DurationVar(&Config.LogSnapshotInterval, "log-snapshot-interval", 0, "print log snapshot interval")
	flag.DurationVar(&Config.NetworkStreamChannelTimeout, "network-stream-channel-timeout", 5*time.Second, "network stream go routine accept new packet timeout")
	flag.DurationVar(&Config.AggregatedLogInterval, "aggregated-log-interval", time.Minute, "print aggregated log messages interval")
	flag.DurationVar(&Config.FullChannelCheckInterval, "full-channel-check-interval", 20*time.Millisecond, "check a full channel interval")
	flag.DurationVar(&Config.FullChannelTimeout, "full-channel-timeout", 5*time.Second, "abandon a full channel after this time")
	flag.DurationVar(&Config.HealthTransactionTimeout, "health-transaction-timeout", 10*time.Second, "return error on health if transaction was not received for this period")

	flag.Parse()
	flag.VisitAll(grabFlagProperties)
	allArgs := "[" + strings.Join(args, ",")[1:] + "]"
	info("All args: %s",allArgs)

	marshal, err := json.Marshal(Config)
	if err != nil {
		fatal("marshal config failed: %v", err)
	}

	V5("V5 mode activated")
	V5("common configuration loaded: %v", string(marshal))

	if Config.Device == "" {
		fatal("device argument must be supplied")
	}

	processors := strings.Split(Config.HarProcessors, ",")
	for i := 0; i < len(processors); i++ {
		processor := processors[i]
		if !supportedProcessors[processor] {
			fatal("invalid har processor specified %v",processor)
		}
	}
	for _,processor := range processors {
		if processor == "s3" && len(Config.S3ExporterBucketName) == 0 {
			fatal("S3 exporter is active and S3 bucket in not defined. Use -s3-bucket-name <my_bucket_name>")
		}
	}
	if Config.Capture != "tshark" && Config.Capture != "httpdump" {
		fatal("invalid capture specified")
	}
	if Config.Hosts == "" {
		info("hosts were not supplied, will capture all IPs on port 80")
	}

	go snapshotTimer()
}
