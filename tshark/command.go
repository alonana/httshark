package tshark

import (
	"bufio"
	"fmt"
	"github.com/alonana/httshark/core"
	"io"
	"os/exec"
	"strings"
)

type JsonProcessor func(data string)

type CommandLine struct {
	lines     chan string
	Processor JsonProcessor
}

func (c *CommandLine) Start() error {
	c.lines = make(chan string, core.Config.ChannelBuffer)

	//TODO: support multiple IPs
	args := fmt.Sprintf("sudo tshark -i %v -f 'tcp port 80 and host %v' -d 'tcp.port==80,http' -Y http -T json",
		core.Config.Device,
		core.Config.Hosts)

	args += " -e frame.time_epoch"
	args += " -e tcp.stream"
	args += " -e http.request"
	args += " -e http.request.method"
	args += " -e http.request.version"
	args += " -e http.request.uri.path"
	args += " -e http.request.uri.query"
	args += " -e http.request.line"
	args += " -e http.file_data"
	args += " -e http.response"
	args += " -e http.response.version"
	args += " -e http.response.code"
	args += " -e http.response.line"

	core.V1("running command: %v", args)
	cmd := exec.Command("sh", "-c", args)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("get command stderr failed: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("get command stdout failed: %v", err)
	}

	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("start command failed: %v", err)
	}

	go c.aggregateJson()
	go c.streamRead(stderr, false)
	go c.streamRead(stdout, true)

	return nil
}

func (c *CommandLine) streamRead(stream io.ReadCloser, collectJson bool) {
	reader := bufio.NewReader(stream)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			core.Fatal("read command output failed: %v", err)
			break
		}
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}
		core.V5("read line: %v", line)
		if collectJson {
			c.lines <- line
		}
	}
}

func (c *CommandLine) aggregateJson() {
	var lines []string
	collect := false
	for {
		line := <-c.lines
		if line == "  {" {
			collect = true
		}
		if collect {
			lines = append(lines, line)
		}
		if line == "  }" {
			data := strings.Join(lines, "")
			core.V5("json data is %v", data)
			c.Processor(data)
			lines = nil
			collect = false
		}
	}
}
