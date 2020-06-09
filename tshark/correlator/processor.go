package correlator

import (
	"github.com/alonana/httshark/core"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type TransactionProcessor func(core.HttpTransaction)

type Processor struct {
	entries     chan interface{}
	requests    map[int]core.HttpRequest
	Processor   TransactionProcessor
	mutex       sync.Mutex
	ticker      *time.Ticker
	waitGroup   sync.WaitGroup
	stopChannel chan bool
	stopped     bool
	Logger      *logrus.Logger
}

func (p *Processor) Start() {
	p.stopped = false
	p.stopChannel = make(chan bool)
	p.requests = make(map[int]core.HttpRequest)
	p.entries = make(chan interface{}, core.Config.ChannelBuffer)
	p.ticker = time.NewTicker(core.Config.ResponseCheckInterval)
	p.waitGroup.Add(1)
	go p.correlate()
}

func (p *Processor) Stop() {
	p.stopChannel <- true
	p.waitGroup.Wait()
}

func (p *Processor) Queue(entry interface{}) {
	p.entries <- entry
}

func (p *Processor) correlate() {
	for !p.stopped {
		select {
		case entry := <-p.entries:
			p.Logger.Trace("got http entry %+v", entry)
			p.updateEntry(&entry)
			break
		case <-p.ticker.C:
			p.checkTimeouts()
			break
		case <-p.stopChannel:
			p.Logger.Debug("correlator processor stopping")
			p.stopped = true
			break
		}
	}
	p.waitGroup.Done()
}

func (p *Processor) updateEntry(entry *interface{}) {
	request, ok := (*entry).(core.HttpRequest)
	if ok {
		p.updateRequest(&request)
		return
	}

	response, ok := (*entry).(core.HttpResponse)
	if ok {
		p.updateResponse(&response)
		return
	}

	p.Logger.Fatal("invalid entry %+v", entry)
}

func (p *Processor) updateRequest(request *core.HttpRequest) {
	p.mutex.Lock()
	p.requests[request.Stream] = *request
	p.mutex.Unlock()
}

func (p *Processor) updateResponse(response *core.HttpResponse) {
	p.mutex.Lock()
	request, exists := p.requests[response.Stream]
	p.mutex.Unlock()

	if !exists {
		p.Logger.Trace("got response without request %+v", response)
		return
	}

	transaction := core.HttpTransaction{
		Request:  request,
		Response: response,
	}

	p.mutex.Lock()
	delete(p.requests, response.Stream)
	p.mutex.Unlock()

	p.Processor(transaction)
}

func (p *Processor) checkTimeouts() {
	p.Logger.Trace("checking timeouts")
	p.mutex.Lock()
	defer p.mutex.Unlock()

	now := time.Now()
	var expired []int
	for stream, request := range p.requests {
		passed := now.Sub(*request.Time)
		if passed > core.Config.ResponseTimeout {
			expired = append(expired, stream)
		}
	}

	if len(expired) == 0 {
		return
	}

	p.Logger.Debug("%v expired requests located", len(expired))
	for i := 0; i < len(expired); i++ {
		stream := expired[i]
		request := p.requests[stream]
		delete(p.requests, stream)
		transaction := core.HttpTransaction{Request: request}
		p.Processor(transaction)
	}
}
