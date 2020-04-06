package tshark

type StdoutJson struct {
	Source Source `json:"_source"`
}

type Source struct {
	Layers Layers `json:"layers"`
}

type Layers struct {
	Time      []string `json:"frame.time_epoch"`
	TcpStream []string `json:"tcp.stream"`
	Data      []string `json:"http.file_data"`

	IsRequest      []string `json:"http.request"`
	RequestMethod  []string `json:"http.request.method"`
	RequestVersion []string `json:"http.request.version"`
	RequestLine    []string `json:"http.request.line"`
	RequestPath    []string `json:"http.request.uri.path"`
	RequestQuery   []string `json:"http.request.uri.query"`

	IsResponse      []string `json:"http.response"`
	ResponseVersion []string `json:"http.response.version"`
	ResponseCode    []string `json:"http.response.code"`
	ResponseLine    []string `json:"http.response.line"`
}
