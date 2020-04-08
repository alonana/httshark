package httpdump

import (
	"errors"
	"github.com/alonana/httshark/core"
	"time"

	"strconv"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/hsiafan/vlog"
)

var logger = vlog.CurrentPackageLogger()

func init() {
	logger.SetAppenders(vlog.NewConsole2Appender())
}

var waitGroup sync.WaitGroup

func listenOneSource(handle *pcap.Handle) chan gopacket.Packet {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()
	return packets
}

// set packet capture filter, by ip and port
func setDeviceFilter(handle *pcap.Handle, filterIP string, filterPort uint16) error {
	var bpfFilter = "tcp"
	if filterPort != 0 {
		bpfFilter += " port " + strconv.Itoa(int(filterPort))
	}
	if filterIP != "" {
		bpfFilter += " ip host " + filterIP
	}
	return handle.SetBPFFilter(bpfFilter)
}

func openSingleDevice(device string, filterIP string, filterPort uint16) (localPackets chan gopacket.Packet, err error) {
	defer func() {
		if msg := recover(); msg != nil {
			switch x := msg.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
			localPackets = nil
		}
	}()
	handle, err := pcap.OpenLive(device, 65536, false, pcap.BlockForever)
	if err != nil {
		return
	}

	if err := setDeviceFilter(handle, filterIP, filterPort); err != nil {
		logger.Warn("set capture filter failed, ", err)
	}
	localPackets = listenOneSource(handle)
	return
}

type TransactionProcessor func(core.HttpTransaction)

var processor TransactionProcessor

func RunHttpDump(p TransactionProcessor) {
	processor = p
	var err error
	packets, err := openSingleDevice(core.Config.Device, core.Config.Hosts, uint16(core.Config.Port))
	if err != nil {
		core.Fatal("listen on device %v failed, error: %w", core.Config.Device, err)
	}

	var handler = &HTTPConnectionHandler{}
	var assembler = newTCPAssembler(handler)
	assembler.filterIP = core.Config.Hosts
	assembler.filterPort = uint16(core.Config.Port)
	var ticker = time.Tick(time.Second * 10)

outer:
	for {
		select {
		case packet := <-packets:
			// A nil packet indicates the end of a pcap file.
			if packet == nil {
				break outer
			}

			// only assembly tcp/ip packets
			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil ||
				packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				continue
			}
			var tcp = packet.TransportLayer().(*layers.TCP)

			assembler.assemble(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

		case <-ticker:
			// flush connections that haven't been activity in the idle time
			assembler.flushOlderThan(time.Now().Add(core.Config.ResponseTimeout))
		}
	}

	assembler.finishAll()
	waitGroup.Wait()
}
