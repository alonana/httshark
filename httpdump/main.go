package httpdump

import (
	"errors"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"strings"
	"time"
)

func listenOneSource(handle *pcap.Handle) chan gopacket.Packet {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := packetSource.Packets()
	return packets
}

// set packet capture filter, by ip and port
func setDeviceFilter(handle *pcap.Handle) error {
	var filter string
	hosts := core.ProduceHosts(core.Config.Hosts).GetHosts()
	if len(hosts) == 1 {
		filter = getHostFilter(hosts[0])
	} else {
		var filters []string
		for i := 0; i < len(hosts); i++ {
			filters = append(filters, getHostFilter(hosts[i]))
		}

		filter = fmt.Sprintf("(%v)", strings.Join(filters, ") or ("))
	}

	core.V2("filter is %v", filter)
	return handle.SetBPFFilter(filter)
}

func getHostFilter(host core.Host) string {
	if len(host.Ip) == 0 {
		return fmt.Sprintf("tcp port %v", host.Port)
	}

	return fmt.Sprintf("tcp port %v and host %v", host.Port, host.Ip)
}

func openSingleDevice(device string) (localPackets chan gopacket.Packet, err error) {
	defer func() {
		if msg := recover(); msg != nil {
			//core.Error("open device recover")
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

	core.V1("open device %v", device)
	handle, err := pcap.OpenLive(device, 65536, true, pcap.BlockForever)
	if err != nil {
		return
	}

	if err := setDeviceFilter(handle); err != nil {
		//core.Fatal("set capture filter failed: %v", err)
	}
	localPackets = listenOneSource(handle)
	return
}

type TransactionProcessor func(core.HttpTransaction)

var processor TransactionProcessor

func RunHttpDump(p TransactionProcessor) {
	processor = p
	var err error
	packets, err := openSingleDevice(core.Config.Device)
	if err != nil {
		//core.Fatal("listen on device %v failed, error: %w", core.Config.Device, err)
	}

	var assembler = newTCPAssembler()
	var ticker = time.Tick(time.Second * 10)

	for {
		core.V2("waiting on http dump channels")
		select {
		case packet := <-packets:
			core.V2("got packet")
			// A nil packet indicates the end of a pcap file.
			if packet == nil {
				//core.Warn("END of PCAP sampling??")
				continue
			}

			// only assembly tcp/ip packets
			if packet.NetworkLayer() == nil || packet.TransportLayer() == nil ||
				packet.TransportLayer().LayerType() != layers.LayerTypeTCP {
				continue
			}
			var tcp = packet.TransportLayer().(*layers.TCP)

			assembler.assemble(packet.NetworkLayer().NetworkFlow(), tcp, packet.Metadata().Timestamp)

		case <-ticker:
			core.V2("flush older")
			assembler.flushOlderThan(time.Now().Add(-core.Config.ResponseTimeout))
		}
	}
}
