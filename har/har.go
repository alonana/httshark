package har

import (
	"fmt"
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

func (e Entry) GetAppId() string {
	return e.Request.AppId.String()
}

