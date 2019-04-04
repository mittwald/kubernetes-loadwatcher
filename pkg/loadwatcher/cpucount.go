package loadwatcher

import "github.com/shirou/gopsutil/cpu"

func determineCPUCount() (cpuCount int32, err error) {
	cpus, err := cpu.Info()
	if err != nil {
		return 0, err
	}

	for i := range cpus {
		cpuCount += cpus[i].Cores
	}

	return
}