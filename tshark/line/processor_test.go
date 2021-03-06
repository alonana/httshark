package line

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"io/ioutil"
	"strings"
	"testing"
)

type DummyTestData struct {
}

func TestEmpty(t *testing.T) {
	p := Processor{}
	p.Start()
	p.Stop()
}

func TestRequest(t *testing.T) {
	runRecord(t, "request")
}

func TestResponse(t *testing.T) {
	runRecord(t, "response")
}

func runRecord(t *testing.T, name string) {
	core.Config.Verbose = 5
	lines := getTestData(t, name)

	var parsed []string

	p := Processor{
		BulkProcessor: func(line string) {
			parsed = append(parsed, line)
		},
	}
	p.Start()

	for i := 0; i < len(lines); i++ {
		p.Queue(lines[i])
	}

	p.Stop()

	if len(parsed) != 1 {
		t.Fatalf("expected one item, but got %v", len(parsed))
	}

	// this is done only to validate the JSON is ok
	var jsonValidation DummyTestData
	err := json.Unmarshal([]byte(parsed[0]), &jsonValidation)
	if err != nil {
		t.Fatal(err)
	}
}

func getTestData(t *testing.T, name string) []string {
	data, err := ioutil.ReadFile(fmt.Sprintf("test_resources/%v.txt", name))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(data), "\n")
}
