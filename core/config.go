package core

import (
	"encoding/json"
	"github.com/namsral/flag"
	"time"
)

type Configuration = struct {
	ChannelBuffer         int
	Verbose               int
	SplitByHost           bool
	Hosts                 string
	DropContentTypes      string
	HarProcessor          string
	Capture               string
	Device                string
	OutputFolder          string
	ResponseTimeout       time.Duration
	ResponseCheckInterval time.Duration
	ExportInterval        time.Duration
}

var Config Configuration

func ParseFlags() {
	flag.IntVar(&Config.ChannelBuffer, "channel-buffer", 1, "channel buffer size")
	flag.IntVar(&Config.Verbose, "verbose", 0, "print verbose information 0=nothing 5=all")
	flag.BoolVar(&Config.SplitByHost, "split-by-host", true, "split output files by the request host")
	flag.StringVar(&Config.OutputFolder, "output-folder", ".", "hal files output folder")
	flag.StringVar(&Config.Hosts, "hosts", ":80", "comma separated list of IP:port to sample e.g. 1.1.1.1:80,2.2.2.2:9090. To sample all hosts on port 9090, use :9090")
	flag.StringVar(&Config.DropContentTypes, "drop-content-type", "image,audio,video", "comma separated list of content type whose body should be removed (case insensitive, using include for match)")
	flag.StringVar(&Config.Device, "device", "", "interface to use sniffing for")
	flag.StringVar(&Config.HarProcessor, "har-processer", "file", "processor of the har file. one of file,memory")
	flag.StringVar(&Config.Capture, "capture", "tshark", "capture engine to use, one of tshark,httpdump")
	flag.DurationVar(&Config.ResponseTimeout, "response-timeout", 5*time.Minute, "timeout for waiting for response")
	flag.DurationVar(&Config.ResponseCheckInterval, "response-check-interval", 10*time.Second, "check timed out responses interval")
	flag.DurationVar(&Config.ExportInterval, "export-interval", 10*time.Second, "export HAL to file interval")

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
	if Config.HarProcessor != "file" && Config.HarProcessor != "memory" {
		Fatal("invalid har processor specified")
	}
	if Config.Capture != "tshark" && Config.Capture != "httpdump" {
		Fatal("invalid capture specified")
	}
	if Config.Hosts == "" {
		Info("hosts were not supplied, will capture all IPs on port 80")
	}
}
