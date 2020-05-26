package tshark

import (
	"bufio"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/exporters"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

type LineProcessor func(line string)
const PACKET_DROP = "Packets received/dropped on interface"
const WARNING = "WARNING"

type CommandLine struct {
	Processor LineProcessor
}

func (c *CommandLine) Start() error {
	args := fmt.Sprintf("sudo dumpcap -i %v -f '%v' -B %v -w - | sudo tshark -r - -Y http -T json",
		core.Config.Device,
		c.getFilter(),
		core.Config.DumpCapBufferSize)


	args += " -e frame.time_epoch"
	args += " -e tcp.stream"
	args += " -e http.request"
	args += " -e http.request.method"
	args += " -e http.request.version"
	args += " -e http.request.full_uri"
	args += " -e http.request.uri"
	args += " -e http.request.line"
	args += " -e http.file_data"
	args += " -e http.response"
	args += " -e http.response.version"
	args += " -e http.response.code"
	args += " -e http.response.line"

	core.Info("running command: %v", args)
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
// build the BPF
func (c *CommandLine) getFilter() string {
	bpf := ""
	hosts := core.ProduceHosts(core.Config.Hosts).GetHosts()
	if len(hosts) == 1 {
		bpf =  "tcp && (" + c.GetHostFilter(hosts[0]) + ")"
	} else {
		var filters []string
		for i := 0; i < len(hosts); i++ {
			filters = append(filters, c.GetHostFilter(hosts[i]))
		}
		bpf = fmt.Sprintf("tcp && ((%v)", strings.Join(filters, ") || (")) + ")"
	}
	core.Info("BPF  = %v",bpf)
	return bpf
}

func (c *CommandLine) GetHostFilter(host core.Host) string {
	if len(host.Ip) == 0 {
		return fmt.Sprintf("port %v", host.Port)
	}

	return fmt.Sprintf("port %v && host %v", host.Port, host.Ip)
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
		} else {
			// this is the error stream - we want to extract a subset of the data into the log
			var packetDropMsg = strings.Index(line,PACKET_DROP) == 0
			if packetDropMsg || strings.Index(line,WARNING) != -1 {
				core.Info("Error stream: %v",line)
				if packetDropMsg {
					left := strings.LastIndex(line,"(")
					right := strings.LastIndex(line,")")
					pct,_ := strconv.Atoi(line[left:right-1])
					pctFloat := float64(pct)
					core.CloudWatchClient.PutMetric("dropped_packets","Percent",
						pctFloat,exporters.NAMESPACE)
				}
			}

		}
	}
}
