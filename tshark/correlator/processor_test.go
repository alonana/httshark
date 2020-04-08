package correlator

import (
	"fmt"
	"github.com/alonana/httshark/core"
	"testing"
	"time"
)

func TestEmpty(t *testing.T) {
	core.Config.ResponseCheckInterval = time.Second
	p := Processor{}
	p.Start()
	p.Stop()
}

func TestTransaction(t *testing.T) {
	core.Config.ResponseCheckInterval = time.Second
	core.Config.ResponseTimeout = time.Second

	var transactions []core.HttpTransaction
	p := Processor{
		Processor: func(transaction core.HttpTransaction) {
			transactions = append(transactions, transaction)
		},
	}
	p.Start()

	now := time.Now()
	stream := 123
	request := core.HttpRequest{
		HttpEntry: core.HttpEntry{
			Time:    &now,
			Stream:  stream,
			Data:    "a",
			Version: "1",
			Headers: []string{"h1"},
		},
		Method: "GET",
		Path:   "/",
		Query:  "b",
	}
	p.Queue(request)

	response := core.HttpResponse{
		HttpEntry: core.HttpEntry{
			Time:    &now,
			Stream:  stream,
			Data:    "b",
			Version: "1",
			Headers: []string{"h2"},
		},
		Code: 200,
	}
	p.Queue(response)
	p.checkTimeouts()
	p.Stop()
	if len(transactions) != 1 {
		t.Fatalf("expected one item, but got %v", len(transactions))
	}

	transaction := transactions[0]
	fmt.Printf("trasaction is %+v\n", transaction)

	if transaction.Response == nil {
		t.Fatalf("response is missing")
	}
}

func TestExpired(t *testing.T) {
	core.Config.ResponseCheckInterval = time.Millisecond
	core.Config.ResponseTimeout = time.Millisecond

	var transactions []core.HttpTransaction
	p := Processor{
		Processor: func(transaction core.HttpTransaction) {
			transactions = append(transactions, transaction)
		},
	}
	p.Start()

	now := time.Now()
	stream := 123
	request := core.HttpRequest{
		HttpEntry: core.HttpEntry{
			Time:    &now,
			Stream:  stream,
			Data:    "a",
			Version: "1",
			Headers: []string{"h1"},
		},
		Method: "GET",
		Path:   "/",
		Query:  "b",
	}
	p.Queue(request)

	time.Sleep(20 * time.Millisecond)

	p.Stop()
	if len(transactions) != 1 {
		t.Fatalf("expected one item, but got %v", len(transactions))
	}

	transaction := transactions[0]
	fmt.Printf("trasaction is %+v\n", transaction)

	if transaction.Response != nil {
		t.Fatalf("response should be empty")
	}
}
