package bulk

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/tshark/types"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HttpProcessor func(interface{})

type StdoutBulkProcessor struct {
	json        chan string
	Processor   HttpProcessor
	waitGroup   sync.WaitGroup
	stopChannel chan bool
	stopped     bool
}

func (p *StdoutBulkProcessor) Start() {
	p.stopChannel = make(chan bool)
	p.json = make(chan string, core.Config.ChannelBuffer)
	p.stopped = false
	p.waitGroup.Add(1)
	go p.parseJson()
}

func (p *StdoutBulkProcessor) Stop() {
	p.stopChannel <- true
	p.waitGroup.Wait()
}

func (p *StdoutBulkProcessor) Queue(data string) {
	p.json <- data
}

func (p *StdoutBulkProcessor) parseJson() {
	for !p.stopped {
		select {
		case data := <-p.json:
			var entry types.Stdout
			err := json.Unmarshal([]byte(data), &entry)
			if err == nil {
				p.convert(&entry)
			} else {
				core.Warn("parse tshark stdout JSON %v failed:%v", data, err)
			}
			break

		case <-p.stopChannel:
			core.V1("stdout bulk processor stopping")
			p.stopped = true
			break
		}
	}

	p.waitGroup.Done()
}

func (p *StdoutBulkProcessor) convert(tsharkJson *types.Stdout) {
	core.V5("json entry is %+v", tsharkJson)
	layers := tsharkJson.Source.Layers

	entryTime, err := p.parseTime(&layers)
	if err != nil {
		core.Warn("parse time in %v failed: %v", tsharkJson, err)
		return
	}

	stream, err := strconv.Atoi(layers.TcpStream[0])
	if err != nil {
		core.Warn("parse tcp stream in %v failed: %v", tsharkJson, err)
		return
	}

	data := ""
	if len(layers.Data) > 0 {
		data = layers.Data[0]
	}

	httpEntry := core.HttpEntry{
		Time:   entryTime,
		Stream: stream,
		Data:   data,
	}

	if len(layers.IsRequest) > 0 {
		httpEntry.Version = layers.RequestVersion[0]
		httpEntry.Headers = layers.RequestLine

		path := "/"
		if len(layers.RequestPath) > 0 {
			path = layers.RequestPath[0]
		}

		query := ""
		if len(layers.RequestQuery) > 0 {
			query = layers.RequestQuery[0]
		}

		request := core.HttpRequest{
			HttpEntry: httpEntry,
			Method:    layers.RequestMethod[0],
			Path:      path,
			Query:     query,
		}

		p.Processor(request)
	} else {
		code, err := strconv.Atoi(layers.ResponseCode[0])
		if err != nil {
			core.Warn("parse response code in %v failed: %v", tsharkJson, err)
			return
		}

		httpEntry.Version = layers.ResponseVersion[0]
		httpEntry.Headers = layers.ResponseLine
		response := core.HttpResponse{
			HttpEntry: httpEntry,
			Code:      code,
		}

		p.Processor(response)
	}

}

func (p *StdoutBulkProcessor) parseTime(layers *types.Layers) (*time.Time, error) {
	epoc := strings.Split(layers.Time[0], ".")
	seconds, err := strconv.ParseInt(epoc[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse time seconds failed: %v", err)
	}
	nanos, err := strconv.ParseInt(epoc[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse time nanos failed: %v", err)
	}

	entryTime := time.Unix(seconds, nanos)
	return &entryTime, nil
}
