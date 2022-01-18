//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"github.com/pkg/errors"
)

// ApplicationSettings is a struct that defines the ApplicationSettings section of the
// configuration.toml file.
type ApplicationSettings struct {
	MobilityProfileThreshold     float64
	MobilityProfileHoldoffMillis float64
	MobilityProfileSlope         float64

	DeviceServiceName string

	DepartedThresholdSeconds     uint
	DepartedCheckIntervalSeconds uint
	AgeOutHours                  uint

	AdjustLastReadOnByOrigin bool
}

// CustomConfig is the struct representation of the individual custom sections
type CustomConfig struct {
	AppSettings ApplicationSettings
	Aliases     map[string]string
}

// ServiceConfig is the struct representation that contains the custom config section
type ServiceConfig struct {
	AppCustom CustomConfig
}

// UpdateFromRaw updates the service's full configuration from raw data received from
// the Service Provider. This function implements the UpdatableConfig interface for ServiceConfig.
func (c *ServiceConfig) UpdateFromRaw(rawConfig interface{}) bool {
	configuration, ok := rawConfig.(*ServiceConfig)
	if !ok {
		return false
	}

	*c = *configuration

	return true
}

var (
	// ErrOutOfRange is returned if a config value is syntactically valid for its type,
	// but otherwise outside of the acceptable range of valid values.
	ErrOutOfRange = errors.New("config value out of range")
)

// NewServiceConfig returns a new ServiceConfig instance with default values.
func NewServiceConfig() ServiceConfig {
	return ServiceConfig{
		AppCustom: CustomConfig{
			Aliases: map[string]string{},
			AppSettings: ApplicationSettings{
				MobilityProfileThreshold:     6,
				MobilityProfileHoldoffMillis: 500,
				MobilityProfileSlope:         -0.008,
				DeviceServiceName:            "device-rfid-llrp",
				DepartedThresholdSeconds:     600,
				DepartedCheckIntervalSeconds: 30,
				AgeOutHours:                  336,
				AdjustLastReadOnByOrigin:     true,
			},
		},
	}
}

// Validate returns nil if the ApplicationSettings are valid,
// or the first validation error it encounters.
func (as ApplicationSettings) Validate() error {
	if as.DepartedThresholdSeconds == 0 {
		return errors.Wrap(ErrOutOfRange, "DepartedThresholdSeconds must be >0")
	}

	if as.DepartedCheckIntervalSeconds == 0 {
		return errors.Wrap(ErrOutOfRange, "DepartedCheckIntervalSeconds must be >0")
	}

	if as.AgeOutHours == 0 {
		return errors.Wrap(ErrOutOfRange, "AgeOutHours must be >0")
	}

	return nil
}
