package httpdump

import (
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/core/aggregated"
	"github.com/google/gopacket/layers"
	"io"
	"time"
)

// NetworkStream tread one-direction tcp data as stream. impl reader closer
type NetworkStream struct {
	window         *ReceiveWindow
	c              chan *layers.TCP
	remain         []byte
	ignore         bool
	closed         bool
	keyDescription string
	opposite       *NetworkStream
}

func newNetworkStream(keyDescription string) *NetworkStream {
	return &NetworkStream{
		window:         newReceiveWindow(64, keyDescription),
		c:              make(chan *layers.TCP, core.Config.NetworkStreamChannelSize),
		keyDescription: keyDescription,
	}
}

func (s *NetworkStream) appendPacket(tcp *layers.TCP) {
	core.V2("stream append packet")
	if s.ignore {
		return
	}
	s.window.insert(tcp)
}

func (s *NetworkStream) confirmPacket(ack uint32) {
	core.V2("stream confirm packet")
	if s.ignore {
		return
	}
	s.window.confirm(ack, s.c)
}

func (s *NetworkStream) finish() {
	close(s.c)
}

func (s *NetworkStream) Read(p []byte) (n int, err error) {
	core.V2("read from %v starting", s.keyDescription)
	for len(s.remain) == 0 {
		timeout := time.NewTimer(core.Config.NetworkStreamChannelTimeout)
		select {
		case packet, ok := <-s.c:
			if !ok {
				core.V2("read from %v EOF", s.keyDescription)
				err = io.EOF
				return
			}
			s.remain = packet.Payload
		case <-timeout.C:
			core.V2("key %v opposite length is %v", s.keyDescription, len(s.opposite.c))
			if len(s.opposite.c) == core.Config.NetworkStreamChannelSize {
				aggregated.Warn("detected stuck stream on, simulating EOF")
				err = io.EOF
				return
			}
			break
		}
	}

	if len(s.remain) > len(p) {
		n = copy(p, s.remain[:len(p)])
		s.remain = s.remain[len(p):]
	} else {
		n = copy(p, s.remain)
		s.remain = nil
	}
	core.V2("read from %v returned %v bytes", s.keyDescription, n)
	return
}

// Close the stream
func (s *NetworkStream) Close() error {
	s.ignore = true
	return nil
}
