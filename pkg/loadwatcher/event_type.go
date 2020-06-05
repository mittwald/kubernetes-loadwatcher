package loadwatcher

type LoadThresholdEvent struct {
	LoadThreshold float64
	Load5         float64
	Load15        float64

	IsHigh  bool
	WasHigh bool
}

func (e *LoadThresholdEvent) ChangedToHigh() bool {
	return e.IsHigh && !e.WasHigh
}

func (e *LoadThresholdEvent) ChangedToLow() bool {
	return !e.IsHigh && e.WasHigh
}
