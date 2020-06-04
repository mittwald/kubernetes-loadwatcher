package loadwatcher

import (
	"context"
	"github.com/shirou/gopsutil/load"
	"k8s.io/klog"
	"time"
)

func (w *Watcher) SetAsHigh(high bool) {
	w.isCurrentlyHigh = high
}

func (w *Watcher) Run(ctx context.Context) (<-chan LoadThresholdEvent, <-chan LoadThresholdEvent, <-chan error) {
	exceeded := make(chan LoadThresholdEvent)
	deceeded := make(chan LoadThresholdEvent)
	errs := make(chan error)
	ticker := time.Tick(w.TickerInterval)

	go func() {
		defer func() {
			close(exceeded)
			close(deceeded)
			close(errs)
		}()

		for {
			select {
			case <-ticker:
				loadStat, err := load.Avg()
				if err != nil {
					errs <- err
				}

				klog.Infof("current state: high_load=%t load5=%.2f load15=%.2f threshold=%.2f", w.isCurrentlyHigh, loadStat.Load5, loadStat.Load15, w.LoadThreshold)

				if loadStat.Load5 >= w.LoadThreshold && !w.isCurrentlyHigh {
					w.isCurrentlyHigh = true
					exceeded <- LoadThresholdEvent{Load5: loadStat.Load5, Load15: loadStat.Load15, LoadThreshold: w.LoadThreshold}
				} else if loadStat.Load5 < w.LoadThreshold && loadStat.Load15 < w.LoadThreshold && w.isCurrentlyHigh {
					w.isCurrentlyHigh = false
					deceeded <- LoadThresholdEvent{Load5: loadStat.Load5, Load15: loadStat.Load15, LoadThreshold: w.LoadThreshold}
				}
			case <-ctx.Done():
				if err := ctx.Err(); err != nil {
					errs <- err
				}
				return
			}
		}
	}()

	return exceeded, deceeded, errs
}
