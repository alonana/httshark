package httpdump

import "strconv"

// Endpoint is one endpoint of a tcp connection
type Endpoint struct {
	ip   string
	port uint16
}

func (p Endpoint) equals(p2 Endpoint) bool {
	return p.ip == p2.ip && p.port == p2.port
}

func (p Endpoint) String() string {
	return p.ip + ":" + strconv.Itoa(int(p.port))
}
