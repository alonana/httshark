package aggregated

import (
	"github.com/alonana/httshark/core"
	"testing"
	"time"
)

func TestEmpty(t *testing.T) {
	core.Config.AggregatedLogInterval = 5 * time.Millisecond
	InitLog()
	Warn("aaa")
	Warn("aaa")
	Warn("aaa")
	Warn("bbb")
	Warn("aaa")
	time.Sleep(100 * time.Millisecond)
	Warn("bbb")
	Warn("ccc")
	Warn("ccc")
	time.Sleep(10 * time.Millisecond)
}
