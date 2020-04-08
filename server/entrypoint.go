package server

import (
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/tshark"
	"github.com/alonana/httshark/tshark/bulk"
	"github.com/alonana/httshark/tshark/correlator"
	"github.com/alonana/httshark/tshark/exporter"
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

	exporterProcessor := exporter.CreateProcessor()
	exporterProcessor.Start()

	correlatorProcessor := correlator.Processor{Processor: exporterProcessor.Queue}
	correlatorProcessor.Start()

	bulkProcessor := bulk.Processor{HttpProcessor: correlatorProcessor.Queue}
	bulkProcessor.Start()

	lineProcessor := line.Processor{BulkProcessor: bulkProcessor.Queue}
	lineProcessor.Start()

	t := tshark.CommandLine{
		Processor: lineProcessor.Queue,
	}
	err := t.Start()
	if err != nil {
		core.Fatal("start command failed: %v", err)
	}

	p.signalsChannel = make(chan os.Signal, 1)
	signal.Notify(p.signalsChannel, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-p.signalsChannel

	core.Info("Termination initiated")
	lineProcessor.Stop()
	bulkProcessor.Stop()
	correlatorProcessor.Stop()
	exporterProcessor.Stop()
	core.Info("Terminating complete")
}
