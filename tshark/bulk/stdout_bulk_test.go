package bulk

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"io/ioutil"
	"testing"
)

func TestEmpty(t *testing.T) {
	s := StdoutBulkProcessor{}
	s.Start()
	s.Stop()
}

func TestRequest(t *testing.T) {
	r := runRecord(t, "request")
	request := r.(core.HttpRequest)
	if request.Method != "GET" {
		t.Fatalf("wrong method %v", request.Method)
	}
}

func TestResponse(t *testing.T) {
	r := runRecord(t, "response")
	response := r.(core.HttpResponse)
	if response.Code != 200 {
		t.Fatalf("wrong code %v", response.Code)
	}
}

func runRecord(t *testing.T, name string) interface{} {
	core.Config.Verbose = 5
	data := getTestData(t, name)

	var parsed []interface{}

	s := StdoutBulkProcessor{
		Processor: func(i interface{}) {
			parsed = append(parsed, i)
		},
	}
	s.Start()
	s.Queue(data)
	s.Stop()

	if len(parsed) != 1 {
		t.Fatalf("expected one item, but got %v", len(parsed))
	}

	fmt.Printf("%+v\n", parsed[0])
	return parsed[0]
}

func getTestData(t *testing.T, name string) string {
	data, err := ioutil.ReadFile(fmt.Sprintf("test_resources/%v.txt", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
