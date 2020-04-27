package httpdump

import (
	"github.com/alonana/httshark/core"
	"github.com/google/gopacket/layers"
	"io"
)

// NetworkStream tread one-direction tcp data as stream. impl reader closer
type NetworkStream struct {
	window *ReceiveWindow
	c      chan *layers.TCP
	remain []byte
	ignore bool
	closed bool
}

func newNetworkStream() *NetworkStream {
	return &NetworkStream{
		window: newReceiveWindow(64),
		c:      make(chan *layers.TCP, core.Config.NetworkStreamChannelSize),
	}
}

func (stream *NetworkStream) appendPacket(tcp *layers.TCP) {
	core.V2("stream append packet")
	if stream.ignore {
		return
	}
	stream.window.insert(tcp)
}

func (stream *NetworkStream) confirmPacket(ack uint32) error {
	core.V2("stream confirm packet")
	if stream.ignore {
		return nil
	}
	return stream.window.confirm(ack, stream.c)
}

func (stream *NetworkStream) finish() {
	close(stream.c)
}

func (stream *NetworkStream) Read(p []byte) (n int, err error) {
	for len(stream.remain) == 0 {
		packet, ok := <-stream.c
		if !ok {
			err = io.EOF
			return
		}
		stream.remain = packet.Payload
	}

	if len(stream.remain) > len(p) {
		n = copy(p, stream.remain[:len(p)])
		stream.remain = stream.remain[len(p):]
	} else {
		n = copy(p, stream.remain)
		stream.remain = nil
	}
	return
}

// Close the stream
func (stream *NetworkStream) Close() error {
	stream.ignore = true
	return nil
}
