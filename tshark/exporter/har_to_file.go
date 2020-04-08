package exporter

import (
	"encoding/json"
	"fmt"
	"github.com/alonana/httshark/core"
	"github.com/alonana/httshark/har"
	"io/ioutil"
	"time"
)

func HarToFile(harData *har.Har) error {
	data, err := json.Marshal(harData)
	if err != nil {
		return fmt.Errorf("marshal har failed: %v", err)
	}

	path := fmt.Sprintf("%v/%v.har", core.Config.OutputFolder, time.Now().Format("2006-01-02T15:04:05"))

	err = ioutil.WriteFile(path, data, 0666)
	if err != nil {
		return fmt.Errorf("write har data to %v failed: %v", path, err)
	}

	core.Info("%v transactions dumped to file %v", len(harData.Log.Entries), path)
	return nil
}
