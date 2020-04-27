package httpdump

import (
	"github.com/alonana/httshark/core"
	"github.com/google/gopacket/layers"
	"time"
)

// TCPConnection hold info for one tcp connection
type TCPConnection struct {
	upStream      *NetworkStream // stream from client to server
	downStream    *NetworkStream // stream from server to client
	clientID      Endpoint       // the client key(by ip and port)
	lastTimestamp time.Time      // timestamp receive last packet
	isHTTP        bool
	key           string
}

// create tcp connection, by the first tcp packet. this packet should from client to server
func newTCPConnection(key string) *TCPConnection {
	connection := &TCPConnection{
		upStream:   newNetworkStream("up " + key),
		downStream: newNetworkStream("down " + key),
		key:        key,
	}

	connection.upStream.opposite = connection.downStream
	connection.downStream.opposite = connection.upStream

	return connection
}

// when receive tcp packet
func (connection *TCPConnection) onReceive(src Endpoint, tcp *layers.TCP, timestamp time.Time) {
	core.V2("connection %v receive", connection.key)
	connection.lastTimestamp = timestamp
	payload := tcp.Payload
	if !connection.isHTTP {
		// skip no-http data
		if !isHTTPRequestData(payload) {
			core.V2("skip non HTTP data")
			return
		}
		// receive first valid http data packet
		connection.clientID = src
		connection.isHTTP = true
	}

	var sendStream, confirmStream *NetworkStream
	if connection.clientID.equals(src) {
		sendStream = connection.upStream
		confirmStream = connection.downStream
	} else {
		sendStream = connection.downStream
		confirmStream = connection.upStream
	}

	sendStream.appendPacket(tcp)

	if tcp.ACK {
		confirmStream.confirmPacket(tcp.Ack)
	}

	// terminate connection
	if tcp.FIN || tcp.RST {
		sendStream.closed = true
	}
}

func (connection *TCPConnection) forceClose() {
	connection.upStream.closed = true
	connection.downStream.closed = true
	connection.finish()

}

func (connection *TCPConnection) closed() bool {
	return connection.upStream.closed && connection.downStream.closed
}

func (connection *TCPConnection) finish() {
	connection.upStream.finish()
	connection.downStream.finish()
}
