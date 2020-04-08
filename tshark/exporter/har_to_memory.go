package exporter

import (
	"github.com/alonana/httshark/har"
)

var MemoryHars []har.Har

func HarToMemory(harData *har.Har) error {
	MemoryHars = append(MemoryHars, *harData)
	return nil
}
