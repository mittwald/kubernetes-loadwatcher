package loadwatcher

import (
	"time"
)

type LoadThresholdEvent struct {
	LoadThreshold float64
	Load5         float64
	Load15        float64
}

type Watcher struct {
	TickerInterval time.Duration
	LoadThreshold  float64
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
