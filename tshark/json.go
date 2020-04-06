package tshark

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"strconv"
	"strings"
	"time"
)

type HttpProcessor func(interface{})

type Json struct {
	json      chan string
	Processor HttpProcessor
}

func (j *Json) Start() {
	j.json = make(chan string, core.Config.ChannelBuffer)
	go j.parseJson()
}

func (j *Json) Queue(data string) {
	j.json <- data
}

func (j *Json) parseJson() {
	for {
		data := <-j.json
		var entry StdoutJson
		err := json.Unmarshal([]byte(data), &entry)
		if err != nil {
			core.Warn("parse JSON %v failed:%v", data, err)
		}
		j.convert(&entry)
	}
}

func (j *Json) convert(tsharkJson *StdoutJson) {
	core.V5("json entry is %+v", tsharkJson)
	layers := tsharkJson.Source.Layers

	entryTime, err := j.parseTime(&layers)
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
		request := core.HttpRequest{
			HttpEntry: httpEntry,
			Method:    layers.RequestMethod[0],
			Path:      layers.RequestPath[0],
			Query:     layers.RequestQuery[0],
		}

		j.Processor(request)
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

		j.Processor(response)
	}

}

func (j *Json) parseTime(layers *Layers) (*time.Time, error) {
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
