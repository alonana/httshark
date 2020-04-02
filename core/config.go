package core

import (
	"encoding/json"
	"github.com/namsral/flag"
)

type Configuration = struct {
	Verbose int
	Hosts   string
}

var Config Configuration

func ParseFlags() {
	flag.IntVar(&Config.Verbose, "verbose", 0, "print verbose information 0=nothing 5=all")
	flag.StringVar(&Config.Hosts, "hosts", "", "comma separated list of IPs to sample. Empty list to sample all hosts")
	flag.Parse()
	marshal, err := json.Marshal(Config)
	if err != nil {
		Fatal("marshal config failed: %v", err)
	}

	V5("V5 mode activated")
	V5("common configuration loaded: %v", string(marshal))
}
