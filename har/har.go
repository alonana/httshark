package har

import (
	"errors"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/core/aggregated"
	"net"
	"net/url"
	"strings"
)

type Cookie struct {
}

type Cache struct {
}

type Timings struct {
	Send    int `json:"send"`
	Wait    int `json:"wait"`
	Receive int `json:"receive"`
}

type Pair struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Content struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type AppIdentifier struct {
	DstIP       string   `json:"dstIp,omitempty"`
	DstPort     int      `json:"dstPort,omitempty"`
}
func (a *AppIdentifier) Empty() {
	a.DstIP = ""
	a.DstPort = 0
}

func (a *AppIdentifier) String() string {
	return fmt.Sprintf("%s_%d", a.DstIP, a.DstPort)
}

type Request struct {
	Method      string         `json:"method"`
	Url         string         `json:"url"`
	HttpVersion string         `json:"httpVersion"`
	Headers     []Pair         `json:"headers"`
	QueryString []Pair         `json:"queryString"`
	Cookies     []Cookie       `json:"cookies"`
	HeadersSize int            `json:"headersSize"`
	BodySize    int            `json:"bodySize"`
	Content     Content        `json:"content"`
	AppId       *AppIdentifier `json:"appId,omitempty"`
}

type Response struct {
	Exists      bool     `json:"exists"`
	Status      int      `json:"status"`
	StatusText  string   `json:"statusText"`
	HttpVersion string   `json:"httpVersion"`
	RedirectUrl string   `json:"redirectURL"`
	Headers     []Pair   `json:"headers"`
	HeadersSize int      `json:"headersSize"`
	BodySize    int      `json:"bodySize"`
	Cookies     []Cookie `json:"cookies"`
	Content     Content  `json:"content"`
}

type Entry struct {
	Started  string   `json:"startedDateTime"`
	Time     int      `json:"time"`
	Request  Request  `json:"request"`
	Response Response `json:"response"`
	Timings  Timings  `json:"timings"`
	Cache    Timings  `json:"cache"`
}

type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Log struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Entries []Entry `json:"entries"`
}

type Har struct {
	Log Log `json:"log"`
}

// handle the case where we have host and port: api-uat.xyz.com:80
func removePortFromHost(host string) string {
	idx := strings.Index(host,":")
	if idx == -1 {
		return host
	} else {
		return host[:idx]
	}
}

func getDomainName(maybeIP string) string {
	if net.ParseIP(maybeIP) != nil {
		fqdn, _ := net.LookupAddr(maybeIP)
		if len(fqdn) > 0 {
			return fqdn[0]
		} else {
			return maybeIP
		}
	} else {
		return maybeIP
	}
}

func (e Entry) GetAppId() string {
	return e.Request.AppId.String()
}

func (e Entry) GetHost() string {
	url, err := url.Parse(e.Request.Url)
	if err != nil {
		aggregated.Warn("Failed to parse url: %v , err: %v",e.Request.Url, core.LimitedError(err))
		host,err := e.getHostByHostHeader()
		if err != nil {
			core.Warn("unable to extract host from %v and unable to find Host header", url)
			return "UNKNOWN"
		} else {
			return getDomainName(removePortFromHost(host))
		}
	} else {
		if len(url.Host) > 0 {
			return getDomainName(removePortFromHost(url.Host))
		} else {
			return "UNKNOWN"
		}
	}
}

func (e Entry) getHostByHostHeader() (string,error) {
	for _, pair := range e.Request.Headers {
		if pair.Name == "Host" {
			return pair.Value,nil
		}
	}
	return "",errors.New("can't find Host header")
}
