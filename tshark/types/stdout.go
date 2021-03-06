package types

type Stdout struct {
	Source Source `json:"_source"`
}

type Source struct {
	Layers Layers `json:"layers"`
}

type Layers struct {
	Time      []string `json:"frame.time_epoch"`
	TcpStream []string `json:"tcp.stream"`
	Data      []string `json:"http.file_data"`

	DstIp    []string `json:"ip.dst"`
	DstPort  []string `json:"tcp.dstport"`

	IsRequest      []string `json:"http.request"`
	RequestMethod  []string `json:"http.request.method"`
	RequestVersion []string `json:"http.request.version"`
	RequestLine    []string `json:"http.request.line"`
	RequestUri     []string `json:"http.request.full_uri"`

	IsResponse      []string `json:"http.response"`
	ResponseVersion []string `json:"http.response.version"`
	ResponseCode    []string `json:"http.response.code"`
	ResponseLine    []string `json:"http.response.line"`
}
