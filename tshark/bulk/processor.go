package bulk

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/tshark/types"
	"github.com/sirupsen/logrus"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HttpProcessor func(interface{})

type Processor struct {
	json          chan string
	HttpProcessor HttpProcessor
	waitGroup     sync.WaitGroup
	stopChannel   chan bool
	stopped       bool
	Logger      *logrus.Logger

}

func (p *Processor) Start() {
	p.stopChannel = make(chan bool)
	p.stopped = false
	p.json = make(chan string, core.Config.ChannelBuffer)
	p.waitGroup.Add(1)
	go p.parseJson()
}

func (p *Processor) Stop() {
	p.stopChannel <- true
	p.waitGroup.Wait()
}

func (p *Processor) Queue(data string) {
	p.json <- data
}

func (p *Processor) parseJson() {
	for !p.stopped {
		select {
		case data := <-p.json:
			var entry types.Stdout
			err := json.Unmarshal([]byte(data), &entry)
			if err == nil {
				p.convert(&entry, data)
			} else {
				p.Logger.Warn(fmt.Sprintf("parse tshark stdout JSON %v failed:%v", data, err))
			}
			break

		case <-p.stopChannel:
			core.V1("bulk processor stopping")
			p.stopped = true
			break
		}
	}

	p.waitGroup.Done()
}

func (p *Processor) convert(tsharkJson *types.Stdout, originalEntry string) {
	core.V5("json entry is %+v", tsharkJson)
	layers := tsharkJson.Source.Layers

	entryTime, err := p.parseTime(&layers)
	if err != nil {
		p.Logger.Warn(fmt.Sprintf("parse time in %+v failed: %v", tsharkJson, err))
		return
	}

	if len(layers.TcpStream) == 0 {
		p.Logger.Warn(fmt.Sprintf("missing tcp stream in %+v", tsharkJson))
		return
	}

	stream, err := strconv.Atoi(layers.TcpStream[0])
	if err != nil {
		p.Logger.Warn(fmt.Sprintf("parse tcp stream in %+v failed: %v", tsharkJson, err))
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
		if len(layers.RequestVersion) > 0 {
			httpEntry.Version = layers.RequestVersion[0]
		}
		httpEntry.Headers = layers.RequestLine

		path := "/"
		query := ""
		if len(layers.RequestUri) > 0 {
			requestUri := layers.RequestUri[0]
			if strings.Contains(requestUri, "?") {
				sections := strings.Split(requestUri, "?")
				path = sections[0]
				query = sections[1]
			} else {
				path = requestUri
			}
		}

		method := ""
		if len(layers.RequestMethod) > 0 {
			method = layers.RequestMethod[0]
		}
		dstIp := layers.DstIp[0]
		dstPort := layers.DstPort[0]
		dstPortInt,err := strconv.Atoi(dstPort)
		if err != nil {
			p.Logger.Warn(fmt.Sprintf("parse dst port in %+v failed: %v", tsharkJson, err))
			return
		}
		ipAndPort := core.HttpIpAndPort{DstIP: dstIp, DstPort: dstPortInt}
		//IpAndPort HttpIpAndPort
		request := core.HttpRequest{
			HttpIpAndPort: ipAndPort,
			HttpEntry: httpEntry,
			Method:    method,
			Path:      path,
			Query:     query,
		}

		p.HttpProcessor(request)
	} else if len(layers.IsResponse) > 0 {
		if len(layers.ResponseCode) == 0 {
			p.Logger.Warn(fmt.Sprintf("missing response code in %+v", tsharkJson))
			return
		}

		code, err := strconv.Atoi(layers.ResponseCode[0])
		if err != nil {
			p.Logger.Warn(fmt.Sprintf("parse response code in %+v failed: %v", tsharkJson, err))
			return
		}

		if len(layers.ResponseVersion) > 0 {
			httpEntry.Version = layers.ResponseVersion[0]
		}
		httpEntry.Headers = layers.ResponseLine
		response := core.HttpResponse{
			HttpEntry: httpEntry,
			Code:      code,
		}

		p.HttpProcessor(response)
	} else {
		core.V5("ignoring not request/response: %v", originalEntry)
	}
}

func (p *Processor) parseTime(layers *types.Layers) (*time.Time, error) {
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
