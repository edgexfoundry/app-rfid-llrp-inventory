package inventory

import "time"

var (
	// todo: these are supposed to be configurable

	AdjustLastReadOnByOrigin = true

	MobilityProfileBaseProfile   = "default"
	MobilityProfileThreshold     = 0.0 // todo
	MobilityProfileHoldoffMillis = 0.0 // todo
	MobilityProfileSlope         = 0.0 // todo

	MobilityProfileSlopeOverridden         = false // todo
	MobilityProfileThresholdOverridden     = false // todo
	MobilityProfileHoldoffMillisOverridden = false // todo

	AggregateDepartedThresholdMillis = int64((1 * time.Hour) / time.Millisecond) // todo
	AgeOutHours                      = 24 * 14                                   // todo

	TagStatsWindowSize = 20
)
