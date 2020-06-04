package loadwatcher

import (
	"time"
)

type Watcher struct {
	TickerInterval time.Duration
	LoadThreshold  float64

	isCurrentlyHigh bool
}

func NewWatcher(loadThreshold int) (*Watcher, error) {
	if loadThreshold == 0 {
		cpuCount, err := determineCPUCount()
		if err != nil {
			return nil, err
		}

		loadThreshold = int(cpuCount)
	}

	return &Watcher{
		LoadThreshold:  float64(loadThreshold),
		TickerInterval: 15 * time.Second,
	}, nil
}
