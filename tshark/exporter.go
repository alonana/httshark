package tshark

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"io/ioutil"
	"strings"
	"sync"
	"time"
)

type Exporter struct {
	input        chan core.HttpTransaction
	transactions []core.HttpTransaction
	mutex        sync.Mutex
}

func (e *Exporter) Start() {
	e.input = make(chan core.HttpTransaction, core.Config.ChannelBuffer)
	go e.aggregate()
	go e.export()
}

func (e *Exporter) Queue(transaction core.HttpTransaction) {
	e.input <- transaction
}

func (e *Exporter) export() {
	tick := time.NewTicker(core.Config.ExportInterval)
	for {
		<-tick.C

		e.mutex.Lock()
		toExport := e.transactions
		e.transactions = nil
		e.mutex.Unlock()

		e.dumpTransactions(toExport)
	}
}

func (e *Exporter) aggregate() {
	for {
		transaction := <-e.input
		core.V5("got transaction %+v", transaction)
		e.mutex.Lock()
		e.transactions = append(e.transactions, transaction)
		e.mutex.Unlock()
	}
}

func (e *Exporter) dumpTransactions(transactions []core.HttpTransaction) {
	if len(transactions) == 0 {
		core.Info("no transactions dumped")
		return
	}

	entries := make([]har.Entry, len(transactions))
	for i := 0; i < len(transactions); i++ {
		entries[i] = e.convert(transactions[i])
	}

	harData := har.Har{
		Log: har.Log{
			Version: "1.2",
			Creator: har.Creator{
				Name:    "httshark",
				Version: "1.0",
			},
			Entries: entries,
		},
	}

	data, err := json.Marshal(harData)
	if err != nil {
		core.Fatal("marshal har failed: %v", err)
	}

	path := fmt.Sprintf("%v/%v.har", core.Config.OutputFolder, time.Now().Format("2006-01-02T15:04:05"))

	err = ioutil.WriteFile(path, data, 0666)
	if err != nil {
		core.Fatal("write har data to %v failed: %v", path, err)
	}

	core.Info("%v transactions dumped to file %v", len(transactions), path)
}

func (e *Exporter) convert(transaction core.HttpTransaction) har.Entry {
	request := transaction.Request
	response := transaction.Response

	duration := 0
	if response != nil {
		duration = int(response.Time.Sub(*request.Time).Milliseconds())
	}

	return har.Entry{
		Started: request.Time.Format("2006-01-02T15:04:05.000Z"),
		Time:    duration,
		Request: har.Request{
			Method:      request.Method,
			Url:         request.Path,
			HttpVersion: request.Version,
			Headers:     e.getHeaders(request.Headers),
		},
	}
}

func (e *Exporter) getHeaders(headers []string) []har.Header {
	harHeaders := make([]har.Header, len(headers))
	for i := 0; i < len(headers); i++ {
		harHeaders[i] = e.getHeader(headers[i])
	}
	return harHeaders
}

func (e *Exporter) getHeader(header string) har.Header {
	position := strings.Index(header, ":")

	if position == -1 {
		return har.Header{
			Name:  header,
			Value: "",
		}
	}

	halHeader := har.Header{
		Name:  strings.TrimSpace(header[0:position]),
		Value: strings.TrimSpace(header[position+1:]),
	}

	return halHeader
}
