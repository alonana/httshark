package server

import "github.com/alonana/httshark/core"

type EntryPoint struct {
}

func (p *EntryPoint) Run() {
	core.ParseFlags()
	core.Info("Starting")
}
