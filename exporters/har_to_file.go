package exporters

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
	hostPrefix := ""
	if core.Config.SplitByHost {
		hostPrefix = harData.Log.Entries[0].GetHost() + "_"
	}

	formattedTime := time.Now().Format("2006-01-02T15:04:05")
	path := fmt.Sprintf("%v/%v%v.har", core.Config.OutputFolder, hostPrefix, formattedTime)

	err = ioutil.WriteFile(path, data, 0666)
	if err != nil {
		return fmt.Errorf("write har data to %v failed: %v", path, err)
	}

	core.Info("%v transactions dumped to file %v", len(harData.Log.Entries), path)
	return nil
}
