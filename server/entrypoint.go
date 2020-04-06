package server

import (
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/tshark"
	"github.com/alonana/httshark/tshark/bulk"
	"github.com/alonana/httshark/tshark/line"
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

	correlator := tshark.Correlator{Processor: exporter.Queue}
	correlator.Start()

	stdoutBulkProcessor := bulk.Processor{HttpProcessor: correlator.Queue}
	stdoutBulkProcessor.Start()

	stdoutLineProcessor := line.Processor{BulkProcessor: stdoutBulkProcessor.Queue}
	stdoutLineProcessor.Start()

	t := tshark.CommandLine{
		Processor: stdoutLineProcessor.Queue,
	}
	err := t.Start()
	if err != nil {
		core.Fatal("start command failed: %v", err)
	}

	p.signalsChannel = make(chan os.Signal, 1)
	signal.Notify(p.signalsChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-p.signalsChannel

	core.Info("Termination initiated")
	stdoutLineProcessor.Stop()
	stdoutBulkProcessor.Stop()
	core.Info("Terminating complete")
}
