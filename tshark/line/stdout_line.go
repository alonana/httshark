package line

import (
	"github.com/alonana/httshark/core"
	"strings"
	"sync"
)

type BulkProcessor func(data string)

type StdoutLineProcessor struct {
	lines         chan string
	BulkProcessor BulkProcessor
	waitGroup     sync.WaitGroup
	stopChannel   chan bool
	stopped       bool
}

func (p *StdoutLineProcessor) Start() {
	p.stopped = false
	p.stopChannel = make(chan bool)
	p.lines = make(chan string, core.Config.ChannelBuffer)
	p.waitGroup.Add(1)
	go p.aggregate()
}

func (p *StdoutLineProcessor) Stop() {
	p.stopChannel <- true
	p.waitGroup.Wait()
}

func (p *StdoutLineProcessor) Queue(line string) {
	p.lines <- line
}

func (p *StdoutLineProcessor) aggregate() {
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
				core.V5("json data is %v", data)
				p.BulkProcessor(data)
				lines = nil
				collect = false
			}
			break

		case <-p.stopChannel:
			core.V1("stdout line processor stopping")
			p.stopped = true
			break
		}
	}
	p.waitGroup.Done()
}
