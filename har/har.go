package har

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Request struct {
	Method      string   `json:"method"`
	Url         string   `json:"url"`
	HttpVersion string   `json:"httpVersion"`
	Headers     []Header `json:"headers"`
}

type Entry struct {
	Started string  `json:"startedDateTime"`
	Time    int     `json:"time"`
	Request Request `json:"request"`
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
