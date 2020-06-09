package line

import (
	"github.com/alonana/httshark/core"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
)

type BulkProcessor func(data string)

type Processor struct {
	lines         chan string
	BulkProcessor BulkProcessor
	waitGroup     sync.WaitGroup
	stopChannel   chan bool
	stopped       bool
	Logger        *logrus.Logger

}

func (p *Processor) Start() {
	p.stopped = false
	p.stopChannel = make(chan bool)
	p.lines = make(chan string, core.Config.ChannelBuffer)
	p.waitGroup.Add(1)
	go p.aggregate()
}

func (p *Processor) Stop() {
	p.stopChannel <- true
	p.waitGroup.Wait()
}

func (p *Processor) Queue(line string) {
	p.lines <- line
}

func (p *Processor) aggregate() {
	var lines []string
	collect := false
	for !p.stopped {
		select {
		case line := <-p.lines:
			if line == "  {" {
				collect = true
			}
			if collect {
				lines = append(lines, strings.TrimSpace(line))
			}
			if line == "  }" {
				data := strings.Join(lines, "")
				p.Logger.Trace("json data is %v", data)
				p.BulkProcessor(data)
				lines = nil
				collect = false
			}
			break

		case <-p.stopChannel:
			p.Logger.Debug("stdout line processor stopping")
			p.stopped = true
			break
		}
	}
	p.waitGroup.Done()
}
