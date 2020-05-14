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
	eofSimulated   bool
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
	if s.ignore || s.eofSimulated {
		return
	}
	s.window.insert(tcp)
}

func (s *NetworkStream) confirmPacket(ack uint32) {
	core.V2("stream confirm packet")
	if s.ignore || s.eofSimulated {
		return
	}
	if !s.waitForChannelSpace() {
		return
	}
	s.window.confirm(ack, s.c)
}

func (s *NetworkStream) waitForChannelSpace() bool {
	startWaitTime := time.Now()
	for len(s.c) >= core.Config.NetworkStreamChannelSize {
		passedTime := time.Now().Sub(startWaitTime)
		if passedTime > core.Config.FullChannelTimeout {
			aggregated.Warn("channel full, abandon data")
			s.ignore = true
			return false
		}
		core.V2("channel is full, waiting before write")
		time.Sleep(core.Config.FullChannelCheckInterval)
	}
	return true
}

func (s *NetworkStream) finish() {
	core.V2("stream %v closing", s.keyDescription)
	close(s.c)
}

func (s *NetworkStream) Read(p []byte) (n int, err error) {
	core.V2("read from %v starting", s.keyDescription)
	lastActiveTime := time.Now()
	for len(s.remain) == 0 {
		timeout := time.NewTimer(core.Config.NetworkStreamChannelTimeout)
		select {
		case packet, ok := <-s.c:
			if !ok {
				core.V2("read from %v EOF", s.keyDescription)
				err = io.EOF
				s.eofSimulated = true
				return
			}
			s.remain = packet.Payload
			lastActiveTime = time.Now()
		case <-timeout.C:
			core.V2("key %v opposite length is %v", s.keyDescription, len(s.opposite.c))
			if len(s.opposite.c) == core.Config.NetworkStreamChannelSize {
				aggregated.Warn("detected stuck stream, simulating EOF")
				err = io.EOF
				s.eofSimulated = true
				return
			}
			nonActive := time.Now().Sub(lastActiveTime)
			if nonActive > core.Config.ResponseTimeout {
				core.V2("non active connection for %v", nonActive)
				aggregated.Warn("simulating EOF on a non active connection")
				err = io.EOF
				s.eofSimulated = true
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
