package exporters

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/core/aggregated"
	"github.com/alonana/httshark/har"
	"github.com/sirupsen/logrus"
	"net/url"
	"strings"
	"sync"
	"time"
)

type HarProcessor func(*har.Har) error

func CreateProcessor(logger *logrus.Logger ) *Processor {
	processor := Processor{Logger: logger}
	processors := strings.Split(core.Config.HarProcessors, ",")
	for i := 0; i < len(processors); i++ {
		name := processors[i]
		var harProcessor HarProcessor
		if name == "sampled-transactions" {
			s := SampleTransactions{}
			go s.init()
			harProcessor = s.Process
		} else if name == "sites-stats" {
			s := SitesStats{Logger: logger}
			go s.init()
			harProcessor = s.Process
		} else if name == "cw-sites-stats" {
			s := PeriodicSiteStats{}
			go s.init()
			harProcessor = s.Process
		} else if name == "file" {
			harProcessor = HarToFile
		} else if name == "s3" {
			s := S3Client{Logger: logger}
			go s.init()
			harProcessor = s.Process
		} else {
			s := TransactionsSizes{}
			go s.init()
			harProcessor = s.Process
		}
		processor.processors = append(processor.processors, harProcessor)
	}
	return &processor
}

type Processor struct {
	Logger              *logrus.Logger
	input               chan core.HttpTransaction
	transactions        []core.HttpTransaction
	mutex               sync.Mutex
	waitGroup           sync.WaitGroup
	stopChannel         chan bool
	stopped             bool
	processors          []HarProcessor
	count               uint64
	lastTransactionTime time.Time
	contentTypesToKeep  []string
}

func (p *Processor) process(harFile *har.Har) error {
	for i := 0; i < len(p.processors); i++ {
		harProcessor := p.processors[i]
		err := harProcessor(harFile)
		if err != nil {
			p.Logger.Warn(fmt.Sprintf("process har failed: %v", err))
		}
	}
	return nil
}

func (p *Processor) Start() {
	p.stopChannel = make(chan bool, 2)
	p.stopped = false
	p.input = make(chan core.HttpTransaction, core.Config.ChannelBuffer)
	p.waitGroup.Add(2)
	p.contentTypesToKeep = strings.Split(core.Config.KeepContentTypes, ",")
	go p.aggregate()
	go p.export()
}

func (p *Processor) Stop() {
	p.stopChannel <- true
	p.stopChannel <- true
	p.waitGroup.Wait()
}

func (p *Processor) Queue(transaction core.HttpTransaction) {
	p.lastTransactionTime = time.Now()
	p.input <- transaction
}

func (p *Processor) CheckHealth() error {
	passed := time.Now().Sub(p.lastTransactionTime)
	if passed > core.Config.HealthTransactionTimeout {
		return fmt.Errorf("transaction was not received for %v", passed)
	}
	return nil
}

func (p *Processor) export() {
	tick := time.NewTicker(core.Config.ExportInterval)
	for !p.stopped {
		select {
		case <-tick.C:

			p.mutex.Lock()
			toExport := p.transactions
			p.transactions = nil
			p.mutex.Unlock()

			p.dumpTransactions(toExport)

		case <-p.stopChannel:
			core.V1("exporter stopping")
			p.stopped = true
			break
		}
	}

	p.waitGroup.Done()
}

func (p *Processor) aggregate() {
	for !p.stopped {
		select {
		case transaction := <-p.input:
			core.V5("got transaction %+v", transaction)
			p.mutex.Lock()
			p.transactions = append(p.transactions, transaction)
			p.mutex.Unlock()

		case <-p.stopChannel:
			core.V1("exporter aggregation stopping")
			p.stopped = true
			break
		}
	}

	p.waitGroup.Done()
}
// we want to ignore Cloud WAF Health Check
func shouldDumpEntry(entry har.Entry) bool {
	if core.Config.IgnoreHealthCheck {
		for i := 0; i < len(entry.Request.Headers); i++ {
			pair := entry.Request.Headers[i]
			if (pair.Name == "Host" && pair.Value == "HOST_FOR_HC") ||
			   (pair.Name == "X-RDWR-HC" && pair.Value == "health check") {
				  return false
			}
		}
	}
	return true
}

func (p *Processor) dumpTransactions(transactions []core.HttpTransaction) {
	if len(transactions) == 0 {
		p.Logger.Info(fmt.Sprintf("no transactions dumped"))
		return
	}
	var entries []har.Entry
	idx := 0
	numOfIgnoredEntries := 0
	for idx < len(transactions) {
		entry := p.convert(transactions[idx])
		if shouldDumpEntry(entry) {
			entries = append(entries,entry)
		} else {
			numOfIgnoredEntries++
		}
		idx++
	}
	harFiles := p.getHarFiles(entries)
	for i := 0; i < len(harFiles); i++ {
		harData := harFiles[i]
		err := p.process(&harData)
		if err != nil {
			p.Logger.Fatal(fmt.Sprintf("marshal har failed: %v", err))
		}
	}
	p.count += uint64(len(transactions) - numOfIgnoredEntries)
	ignoredPct := int((float64(numOfIgnoredEntries)/float64(len(transactions))) * 100)
	p.Logger.Info(fmt.Sprintf("%v total transactions dumped so far. [current cycle: %v, ignored transactions: %v (~ %v%%)]",
		p.count,
		len(transactions),
		numOfIgnoredEntries,ignoredPct))
}

func (p *Processor) getHarFiles(entries []har.Entry) []har.Har {
	if !core.Config.SplitByHost {
		harData := p.getHarFile(entries)
		return []har.Har{*harData}
	}

	appIdEntriesMap := make(map[string][]har.Entry)
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		appId := entry.GetAppId()
		appIdEntries := appIdEntriesMap[appId]
		appIdEntries = append(appIdEntries, entry)
		appIdEntriesMap[appId] = appIdEntries
	}

	var files []har.Har
	for _, appIdEntries := range appIdEntriesMap {
		harData := p.getHarFile(appIdEntries)
		files = append(files, *harData)
	}
	return files
}

func (p *Processor) getHarFile(entries []har.Entry) *har.Har {
	return &har.Har{
		Log: har.Log{
			Version: "1.2",
			Creator: har.Creator{
				Name:    "httshark",
				Version: "1.0",
			},
			Entries: entries,
		},
	}
}

func (p *Processor) convert(transaction core.HttpTransaction) har.Entry {
	request := transaction.Request
	response := transaction.Response

	duration := 0
	harResponse := har.Response{
		Exists:      false,
		Cookies:     make([]har.Cookie, 0),
		HeadersSize: -1,
		Content: har.Content{
			Size:     0,
			MimeType: "",
			Text:     "",
		},
	}
	if response != nil {
		harResponse.Exists = true
		duration = int(response.Time.Sub(*request.Time).Milliseconds())
		harResponse.Status = response.Code
		harResponse.Headers = p.getHeaders(response.Headers)
		harResponse.HeadersSize = p.getHeadersSize(response.Headers)
		harResponse.HttpVersion = response.Version
		harResponse.BodySize = len(response.Data)
		harResponse.Content.Size = len(response.Data)
		harResponse.Content.Text = response.Data

		if !p.shouldKeep(harResponse.Headers) {
			harResponse.Content.Text = ""
		}
	}

	harRequest := har.Request{
		AppId:       &har.AppIdentifier{DstIP: request.HttpIpAndPort.DstIP, DstPort: request.HttpIpAndPort.DstPort},
		Method:      request.Method,
		Url:         request.Path,
		HttpVersion: request.Version,
		Headers:     p.getHeaders(request.Headers),
		QueryString: p.getQueryString(request.Query),
		Cookies:     make([]har.Cookie, 0),
		HeadersSize: p.getHeadersSize(request.Headers),
		BodySize:    len(request.Data),
		Content: har.Content{
			Size:     len(request.Data),
			MimeType: "",
			Text:     request.Data,
		},
	}
	if !p.shouldKeep(harRequest.Headers) {
		harResponse.Content.Text = ""
	}


	return har.Entry{
		Started:  request.Time.Format("2006-01-02T15:04:05.000Z"),
		Time:     duration,
		Request:  harRequest,
		Response: harResponse,
	}
}

func (p *Processor) getHeaders(headers []string) []har.Pair {
	harHeaders := make([]har.Pair, len(headers))
	for i := 0; i < len(headers); i++ {
		harHeaders[i] = p.getHeader(headers[i])
	}
	return harHeaders
}

func (p *Processor) getHeadersSize(headers []string) int {
	size := 0
	for i := 0; i < len(headers); i++ {
		size += len(headers[i])
	}
	return size
}

func (p *Processor) getHeader(header string) har.Pair {
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

func (p *Processor) getQueryString(query string) []har.Pair {
	queryString := make([]har.Pair, 0)
	values, err := url.ParseQuery(query)
	if err != nil {
		core.V5("parse query string %s failed: %v", query, err)
		aggregated.Warn("parse query string failed: %v", core.LimitedError(err))
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


func (p *Processor) shouldKeep(headers []har.Pair) bool {
	for i := 0; i < len(headers); i++ {
		header := headers[i]
		if strings.ToLower(header.Name) == "content-type" && p.shouldKeepContentType(header.Value) {
			return true
		}
	}
	return false
}
/*
func (p *Processor) isDrop(headers []har.Pair) bool {
	for i := 0; i < len(headers); i++ {
		header := headers[i]
		if strings.ToLower(header.Name) == "content-type" && p.isDropContentType(header.Value) {
			return true
		}
	}
	return false
}
*/


/*
func (p *Processor) isDropContentType(value string) bool {
	if len(core.Config.DropContentTypes) == 0 {
		return false
	}

	value = strings.ToLower(value)
	types := strings.Split(core.Config.DropContentTypes, ",")
	for i := 0; i < len(types); i++ {
		if strings.Contains(value, types[i]) {
			return true
		}
	}
	return false
}
*/

func (p *Processor) shouldKeepContentType(value string) bool {
	value = strings.ToLower(value)
	for i := 0; i < len(p.contentTypesToKeep); i++ {
		if strings.Contains(value, p.contentTypesToKeep[i]) {
			return true
		}
	}
	return false
}
