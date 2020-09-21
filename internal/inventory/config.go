//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

type ApplicationSettingsType struct {
	DeviceServiceName  string
	DeviceServiceURL   string
	MetadataServiceURL string

	AdjustLastReadOnByOrigin     bool
	DepartedThresholdSeconds     int
	DepartedCheckIntervalSeconds int
	AgeOutHours                  int
}

type ConsulConfig struct {
	ApplicationSettings ApplicationSettingsType
	Aliases             map[string]string
}
