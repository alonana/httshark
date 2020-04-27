package httpdump

import (
	"github.com/alonana/httshark/core"
	"github.com/google/gopacket/layers"
)

// ReceiveWindow simulate tcp receive window
type ReceiveWindow struct {
	size           int
	start          int
	buffer         []*layers.TCP
	lastAck        uint32
	expectBegin    uint32
	keyDescription string
}

func newReceiveWindow(initialSize int, keyDescription string) *ReceiveWindow {
	buffer := make([]*layers.TCP, initialSize)
	return &ReceiveWindow{
		buffer:         buffer,
		keyDescription: keyDescription,
	}
}

func (w *ReceiveWindow) destroy() {
	w.size = 0
	w.start = 0
	w.buffer = nil
}

func (w *ReceiveWindow) insert(packet *layers.TCP) {
	if w.expectBegin != 0 && compareTCPSeq(w.expectBegin, packet.Seq+uint32(len(packet.Payload))) >= 0 {
		// dropped
		return
	}

	if len(packet.Payload) == 0 {
		//ignore empty data packet
		return
	}

	idx := w.size
	for ; idx > 0; idx-- {
		index := (idx - 1 + w.start) % len(w.buffer)
		prev := w.buffer[index]
		result := compareTCPSeq(prev.Seq, packet.Seq)
		if result == 0 {
			// duplicated
			return
		}
		if result < 0 {
			// insert at index
			break
		}
	}

	if w.size == len(w.buffer) {
		w.expand()
	}

	if idx == w.size {
		// append at last
		index := (idx + w.start) % len(w.buffer)
		w.buffer[index] = packet
	} else {
		// insert at index
		for i := w.size - 1; i >= idx; i-- {
			next := (i + w.start + 1) % len(w.buffer)
			current := (i + w.start) % len(w.buffer)
			w.buffer[next] = w.buffer[current]
		}
		index := (idx + w.start) % len(w.buffer)
		w.buffer[index] = packet
	}

	w.size++
}

// send confirmed packets to reader, when receive ack
func (w *ReceiveWindow) confirm(ack uint32, c chan *layers.TCP) {
	idx := 0
	core.V2("confirm window size %v", w.size)
	for ; idx < w.size; idx++ {
		index := (idx + w.start) % len(w.buffer)
		packet := w.buffer[index]
		result := compareTCPSeq(packet.Seq, ack)
		if result >= 0 {
			break
		}
		w.buffer[index] = nil
		newExpect := packet.Seq + uint32(len(packet.Payload))
		if w.expectBegin != 0 {
			diff := compareTCPSeq(w.expectBegin, packet.Seq)
			if diff > 0 {
				duplicatedSize := w.expectBegin - packet.Seq
				if duplicatedSize < 0 {
					duplicatedSize += maxTCPSeq
				}
				if duplicatedSize >= uint32(len(packet.Payload)) {
					continue
				}
				packet.Payload = packet.Payload[duplicatedSize:]
			} else if diff < 0 {
				core.V2("we lose packet here")
			}
		}
		core.V2("key %v add packet to channel start", w.keyDescription)
		c <- packet
		core.V2("key %v add packet to channel done len %v", w.keyDescription, len(packet.Payload))
		w.expectBegin = newExpect
	}
	w.start = (w.start + idx) % len(w.buffer)
	w.size = w.size - idx
	core.V2("confirm loop done")
	if compareTCPSeq(w.lastAck, ack) < 0 || w.lastAck == 0 {
		w.lastAck = ack
	}
}

func (w *ReceiveWindow) expand() {
	buffer := make([]*layers.TCP, len(w.buffer)*2)
	end := w.start + w.size
	if end < len(w.buffer) {
		copy(buffer, w.buffer[w.start:w.start+w.size])
	} else {
		copy(buffer, w.buffer[w.start:])
		copy(buffer[len(w.buffer)-w.start:], w.buffer[:end-len(w.buffer)])
	}
	w.start = 0
	w.buffer = buffer
}

// compare two tcp sequences, if seq1 is earlier, return num < 0, if seq1 == seq2, return 0, else return num > 0
func compareTCPSeq(seq1, seq2 uint32) int {
	if seq1 < tcpSeqWindow && seq2 > maxTCPSeq-tcpSeqWindow {
		return int(seq1 + maxTCPSeq - seq2)
	} else if seq2 < tcpSeqWindow && seq1 > maxTCPSeq-tcpSeqWindow {
		return int(seq1 - (maxTCPSeq + seq2))
	}
	return int(int32(seq1 - seq2))
}
