//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

type ApplicationSettings struct {
	AdjustLastReadOnByOrigin     bool
	DepartedThresholdSeconds     int
	DepartedCheckIntervalSeconds int
	AgeOutHours                  int

	MobilityProfileThreshold     float64
	MobilityProfileHoldoffMillis float64
	MobilityProfileSlope         float64

	DeviceServiceName  string
	DeviceServiceURL   string
	MetadataServiceURL string
}

type ConsulConfig struct {
	ApplicationSettings ApplicationSettings
	Aliases             map[string]string
}
