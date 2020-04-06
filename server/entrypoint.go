package server

import (
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/tshark"
	"os"
	"os/signal"
	"syscall"
)

type EntryPoint struct {
	signalsChannel chan os.Signal
}

func (p *EntryPoint) Run() {
	core.ParseFlags()
	core.Info("Starting")

	exporter := tshark.Exporter{}
	exporter.Start()

	correlator := tshark.Correlator{
		Processor: exporter.Queue,
	}
	correlator.Start()

	jsonParser := tshark.Json{
		Processor: correlator.Queue,
	}
	jsonParser.Start()

	t := tshark.CommandLine{
		Processor: jsonParser.Queue,
	}
	err := t.Start()
	if err != nil {
		core.Fatal("start command failed: %v", err)
	}

	p.signalsChannel = make(chan os.Signal, 1)
	signal.Notify(p.signalsChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-p.signalsChannel

	core.Info("Terminating")
}
