package tshark

import (
	"github.com/alonana/httshark/core"
)

type Exporter struct {
	transactions chan core.HttpTransaction
}

func (e *Exporter) Start() {
	e.transactions = make(chan core.HttpTransaction, core.Config.ChannelBuffer)
	go e.export()
}

func (e *Exporter) Queue(transaction core.HttpTransaction) {
	e.transactions <- transaction
}

func (e *Exporter) export() {
	for {
		transaction := <-e.transactions
		core.V5("got transaction %+v", transaction)
	}
}
