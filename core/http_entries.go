package core

import "time"

type HttpEntry struct {
	Time    *time.Time
	Stream  int
	Data    string
	Version string
	Headers []string
}

type HttpIpAndPort struct {
	DstIP       string
	DstPort     int
}

type HttpRequest struct {
	HttpEntry
	Method        string
	Path          string
	Query         string
	HttpIpAndPort HttpIpAndPort
}

type HttpResponse struct {
	HttpEntry
	Code int
}

type HttpTransaction struct {
	Request  HttpRequest
	Response *HttpResponse
}
