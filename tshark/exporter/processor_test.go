package exporter

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"testing"
	"time"
)

func TestEmpty(t *testing.T) {
	core.Config.Verbose = 5
	core.Config.ExportInterval = time.Millisecond
	p := CreateProcessor()
	p.Start()
	p.Stop()
}

func TestTransactions(t *testing.T) {
	core.Config.Verbose = 5
	core.Config.ExportInterval = time.Millisecond
	core.Config.DropContentTypes = "image,audio,video"
	p := CreateProcessor()
	p.Start()

	now := time.Now()
	transaction := core.HttpTransaction{
		Request: core.HttpRequest{
			HttpEntry: core.HttpEntry{
				Time:    &now,
				Stream:  0,
				Data:    "data to keep",
				Version: "1",
				Headers: []string{"content-type: x-application-form"},
			},
			Method: "GET",
			Path:   "/",
			Query:  "",
		},
		Response: &core.HttpResponse{
			HttpEntry: core.HttpEntry{
				Time:    &now,
				Stream:  0,
				Data:    "data to drop",
				Version: "",
				Headers: []string{"content-type: image/png"},
			},
			Code: 0,
		},
	}
	p.Queue(transaction)

	time.Sleep(20 * time.Millisecond)

	if len(MemoryHars) != 1 {
		t.Fatalf("expected one item, but got %v", len(MemoryHars))
	}

	harData := MemoryHars[0]
	fmt.Printf("%+v\n", harData)

	entry := harData.Log.Entries[0]
	if entry.Request.Content.Text == "" {
		t.Fatalf("request content should not had been dropped")
	}
	if entry.Response.Content.Text != "" {
		t.Fatalf("response content should had been dropped")
	}

	p.Stop()
}
