package core

import (
	"encoding/json"
	"github.com/namsral/flag"
	"time"
)

type Configuration = struct {
	ChannelBuffer   int
	Verbose         int
	Hosts           string
	Device          string
	ResponseTimeout time.Duration
}

var Config Configuration

func ParseFlags() {
	flag.IntVar(&Config.ChannelBuffer, "channel-buffer", 10, "channel buffer size")
	flag.IntVar(&Config.Verbose, "verbose", 0, "print verbose information 0=nothing 5=all")
	flag.StringVar(&Config.Hosts, "hosts", "", "comma separated list of IPs to sample. Empty list to sample all hosts")
	flag.StringVar(&Config.Device, "device", "", "interface to use sniffing for")
	flag.DurationVar(&Config.ResponseTimeout, "response-timeout", 5*time.Minute, "timeout for waiting for response")

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
	if Config.Hosts == "" {
		Fatal("hosts argument must be supplied")
	}
}
