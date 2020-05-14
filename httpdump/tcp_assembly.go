package httpdump

import (
	"bytes"
	"github.com/alonana/httshark/core"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"sync"
	"time"
)

// gopacket provide a tcp connection, however it split one tcp connection into two stream.
// So it is hard to match http request and response. we make our own connection here

const maxTCPSeq uint32 = 0xFFFFFFFF
const tcpSeqWindow = 0x0000FFFF

// TCPAssembler do tcp package assemble
type TCPAssembler struct {
	connectionDict map[string]*TCPConnection
	lock           sync.Mutex
}

func newTCPAssembler() *TCPAssembler {
	return &TCPAssembler{
		connectionDict: map[string]*TCPConnection{},
	}
}

func (assembler *TCPAssembler) assemble(flow gopacket.Flow, tcp *layers.TCP, timestamp time.Time) {
	core.V2("received packet")
	src := Endpoint{ip: flow.Src().String(), port: uint16(tcp.SrcPort)}
	dst := Endpoint{ip: flow.Dst().String(), port: uint16(tcp.DstPort)}

	srcString := src.String()
	dstString := dst.String()
	var key string
	if srcString < dstString {
		key = srcString + "-" + dstString
	} else {
		key = dstString + "-" + srcString
	}

	var createNewConn = tcp.SYN && !tcp.ACK || isHTTPRequestData(tcp.Payload)
	connection := assembler.retrieveConnection(src, dst, key, createNewConn)
	if connection == nil {
		core.V2("connection %v not located", key)
		return
	}

	connection.onReceive(src, tcp, timestamp)

	if connection.closed() {
		core.V2("%v assembly - closing", key)
		assembler.deleteConnection(key)
		connection.finish()
	}
}

// get connection this packet belong to; create new one if is new connection
func (assembler *TCPAssembler) retrieveConnection(src, dst Endpoint, key string, init bool) *TCPConnection {
	assembler.lock.Lock()
	defer assembler.lock.Unlock()
	connection := assembler.connectionDict[key]
	if connection == nil {
		if init {
			connection = newTCPConnection(key)
			assembler.connectionDict[key] = connection
			newHttpTrafficHandler(key, src, dst, connection)
			core.V2("creating connection %v", key)
		}
	}
	return connection
}

// remove connection (when is closed or timeout)
func (assembler *TCPAssembler) deleteConnection(key string) {
	assembler.lock.Lock()
	defer assembler.lock.Unlock()
	delete(assembler.connectionDict, key)
}

// flush timeout connections
func (assembler *TCPAssembler) flushOlderThan(time time.Time) {
	var connections []*TCPConnection
	assembler.lock.Lock()
	for _, connection := range assembler.connectionDict {
		if connection.lastTimestamp.Before(time) {
			connections = append(connections, connection)
		}
	}
	for _, connection := range connections {
		delete(assembler.connectionDict, connection.key)
	}
	assembler.lock.Unlock()

	for _, connection := range connections {
		connection.forceClose()
	}
}

var httpMethods = map[string]bool{"GET": true, "POST": true, "PUT": true, "DELETE": true, "HEAD": true,
	"TRACE": true, "OPTIONS": true, "PATCH": true}

// if is first http request packet
func isHTTPRequestData(body []byte) bool {
	if len(body) < 8 {
		return false
	}
	data := body[0:8]
	idx := bytes.IndexByte(data, byte(' '))
	if idx < 0 {
		return false
	}

	method := string(data[:idx])
	return httpMethods[method]
}
