package tshark

import (
	"github.com/alonana/httshark/core"
)

type TransactionProcessor func(core.HttpTransaction)

type Correlator struct {
	entries   chan interface{}
	requests  map[int]core.HttpRequest
	Processor TransactionProcessor
}

func (c *Correlator) Start() {
	c.requests = make(map[int]core.HttpRequest)
	c.entries = make(chan interface{}, core.Config.ChannelBuffer)
	go c.correlate()
}

func (c *Correlator) Queue(entry interface{}) {
	c.entries <- entry
}

func (c *Correlator) correlate() {
	for {
		entry := <-c.entries
		core.V5("got http entry %+v", entry)
		c.updateEntry(&entry)
	}
}

func (c *Correlator) updateEntry(entry *interface{}) {
	request, ok := (*entry).(core.HttpRequest)
	if ok {
		c.updateRequest(&request)
		return
	}

	response, ok := (*entry).(core.HttpResponse)
	if ok {
		c.updateResponse(&response)
		return
	}

	core.Fatal("invalid entry %+v", entry)
}

func (c *Correlator) updateRequest(request *core.HttpRequest) {
	c.requests[request.Stream] = *request
}

func (c *Correlator) updateResponse(response *core.HttpResponse) {
	request, exists := c.requests[response.Stream]
	if !exists {
		core.Warn("got response without request %+v", response)
		return
	}

	transaction := core.HttpTransaction{
		Request:  request,
		Response: *response,
	}

	delete(c.requests, response.Stream)

	c.Processor(transaction)
}
