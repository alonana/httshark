package tshark

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"io/ioutil"
	"net/url"
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
	harResponse := har.Response{
		Cookies:     make([]har.Cookie, 0),
		HeadersSize: -1,
		Content: har.Content{
			Size:     0,
			MimeType: "",
			Text:     "",
		},
	}
	if response != nil {
		duration = int(response.Time.Sub(*request.Time).Milliseconds())
		harResponse.Status = response.Code
		harResponse.Headers = e.getHeaders(response.Headers)
		harResponse.HttpVersion = response.Version
		harResponse.BodySize = len(response.Data)
		harResponse.Content.Size = len(response.Data)
		harResponse.Content.Text = response.Data
	}

	return har.Entry{
		Started: request.Time.Format("2006-01-02T15:04:05.000Z"),
		Time:    duration,
		Request: har.Request{
			Method:      request.Method,
			Url:         request.Path,
			HttpVersion: request.Version,
			Headers:     e.getHeaders(request.Headers),
			QueryString: e.getQueryString(request.Query),
			Cookies:     make([]har.Cookie, 0),
			HeadersSize: -1,
			BodySize:    len(request.Data),
			Content: har.Content{
				Size:     len(request.Data),
				MimeType: "",
				Text:     request.Data,
			},
		},
		Response: harResponse,
	}
}

func (e *Exporter) getHeaders(headers []string) []har.Pair {
	harHeaders := make([]har.Pair, len(headers))
	for i := 0; i < len(headers); i++ {
		harHeaders[i] = e.getHeader(headers[i])
	}
	return harHeaders
}

func (e *Exporter) getHeader(header string) har.Pair {
	position := strings.Index(header, ":")

	if position == -1 {
		return har.Pair{
			Name:  header,
			Value: "",
		}
	}

	halHeader := har.Pair{
		Name:  strings.TrimSpace(header[0:position]),
		Value: strings.TrimSpace(header[position+1:]),
	}

	return halHeader
}

func (e *Exporter) getQueryString(query string) []har.Pair {
	queryString := make([]har.Pair, 0)
	values, err := url.ParseQuery(query)
	if err != nil {
		core.Warn("parse query string %v failed: %v", query, err)
		return queryString
	}

	for k, v := range values {
		pair := har.Pair{
			Name:  k,
			Value: v[0],
		}
		queryString = append(queryString, pair)
	}

	return queryString
}
