package core

import (
	"encoding/json"
	"github.com/namsral/flag"
	"time"
)

type Configuration = struct {
	ChannelBuffer         int
	Port                  int
	Verbose               int
	Hosts                 string
	DropContentTypes      string
	HarProcessor          string
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
	flag.IntVar(&Config.Port, "port", 80, "filter packets for this port")
	flag.StringVar(&Config.OutputFolder, "output-folder", ".", "hal files output folder")
	flag.StringVar(&Config.Hosts, "hosts", "", "comma separated list of IPs to sample. Empty list to sample all hosts")
	flag.StringVar(&Config.DropContentTypes, "drop-content-type", "image,audio,video", "comma separated list of content type whose body should be removed (case insensitive, using include for match)")
	flag.StringVar(&Config.Device, "device", "", "interface to use sniffing for")
	flag.StringVar(&Config.HarProcessor, "har-processer", "file", "processor of the har file. one of file,memory")
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
	if Config.Hosts == "" {
		Fatal("hosts argument must be supplied")
	}
}
