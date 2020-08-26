/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

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

	// NOTE: If you change the default values, please update the README.md
	DepartedThresholdSeconds     = 30
	DepartedCheckIntervalSeconds = 10
	AgeOutHours                  = 336

	TagStatsWindowSize = 20
)
