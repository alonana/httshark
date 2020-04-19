package tshark

import (
	"bufio"
	"fmt"
	"github.com/alonana/httshark/core"
	"io"
	"os/exec"
	"strings"
)

type LineProcessor func(line string)

type CommandLine struct {
	Processor LineProcessor
}

func (c *CommandLine) Start() error {
	args := fmt.Sprintf("sudo tshark -i %v -f '%v'  -Y http -T json",
		core.Config.Device,
		c.getFilter())

	args += " -e frame.time_epoch"
	args += " -e tcp.stream"
	args += " -e http.request"
	args += " -e http.request.method"
	args += " -e http.request.version"
	args += " -e http.request.uri"
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

	go c.streamRead(stderr, false)
	go c.streamRead(stdout, true)

	return nil
}

func (c *CommandLine) getFilter() string {
	hosts := core.ProduceHosts(core.Config.Hosts).GetHosts()
	if len(hosts) == 1 {
		return c.GetHostFilter(hosts[0])
	}

	var filters []string
	for i := 0; i < len(hosts); i++ {
		filters = append(filters, c.GetHostFilter(hosts[i]))
	}

	return fmt.Sprintf("(%v)", strings.Join(filters, ") or ("))
}

func (c *CommandLine) GetHostFilter(host core.Host) string {
	if len(host.Ip) == 0 {
		return fmt.Sprintf("tcp port %v", host.Port)
	}

	return fmt.Sprintf("tcp port %v and host %v", host.Port, host.Ip)
}

func (c *CommandLine) getPortsFilter() string {
	if len(core.Config.Hosts) == 0 {
		return ""
	}

	if !strings.Contains(core.Config.Hosts, ",") {
		return fmt.Sprintf("and host %v", core.Config.Hosts)
	}

	filter := strings.Join(strings.Split(core.Config.Hosts, ","), " or host ")
	return fmt.Sprintf("and (host %v)", filter)
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
			c.Processor(line)
		}
	}
}
