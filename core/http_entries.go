package core

import "time"

type HttpEntry struct {
	Time    *time.Time
	Stream  int
	Data    string
	Version string
	Headers []string
}

type HttpRequest struct {
	HttpEntry
	Method string
	Path   string
	Query  string
}

type HttpResponse struct {
	HttpEntry
	Code int
}

type HttpTransaction struct {
	Request  HttpRequest
	Response HttpResponse
}
