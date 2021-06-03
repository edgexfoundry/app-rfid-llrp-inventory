//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
	"strconv"
)

// ApplicationSettings is a struct that defines the ApplicationSettings section of the
// configuration.toml file.
type ApplicationSettings struct {
	MobilityProfileThreshold     float64
	MobilityProfileHoldoffMillis float64
	MobilityProfileSlope         float64

	DeviceServiceName  string
	DeviceServiceURL   string
	MetadataServiceURL string

	DepartedThresholdSeconds     uint
	DepartedCheckIntervalSeconds uint
	AgeOutHours                  uint

	AdjustLastReadOnByOrigin bool
}

// WriteableConfig is a struct representation of the Writeable section of the configuration.toml file.
type WriteableConfig struct {
	LogLevel string
}

// ConsulConfig is the struct representation of all of the sections from the configuration.toml
// that we are interested in syncing with Consul.
type ConsulConfig struct {
	Writable            WriteableConfig
	ApplicationSettings ApplicationSettings
	Aliases             map[string]string
}

var (
	// ErrUnexpectedConfigItems is returned when the input configuration map has extra keys
	// and values that are left over after parsing is complete
	ErrUnexpectedConfigItems = errors.New("unexpected config items")
	// ErrMissingRequiredKey is returned when we are unable to parse the value for a config key
	ErrMissingRequiredKey = errors.New("missing required key")
	// ErrOutOfRange is returned if a config value is syntactically valid for its type,
	// but otherwise outside of the acceptable range of valid values.
	ErrOutOfRange = errors.New("config value out of range")
)

// NewConsulConfig returns a new ConsulConfig instance with default values.
func NewConsulConfig() ConsulConfig {
	return ConsulConfig{
		Aliases: map[string]string{},
		Writable: WriteableConfig{
			LogLevel: "INFO",
		},
		ApplicationSettings: ApplicationSettings{
			MobilityProfileThreshold:     6,
			MobilityProfileHoldoffMillis: 500,
			MobilityProfileSlope:         -0.008,
			DeviceServiceName:            "edgex-device-llrp",
			DeviceServiceURL:             "http://edgex-device-llrp:49989/",
			MetadataServiceURL:           "http://edgex-core-metadata:48081/",
			DepartedThresholdSeconds:     600,
			DepartedCheckIntervalSeconds: 30,
			AgeOutHours:                  336,
			AdjustLastReadOnByOrigin:     true,
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

// ParseConsulConfig returns a new ConsulConfig
// with settings parsed from the given map,
// merged with default settings for missing value.
//
// It returns a parsing error if a given key's value cannot be parsed,
// an error wrapping ErrMissingRequiredKey if a required key is missing,
// and an error wrapping ErrUnexpectedConfigItems if the map has unknown config keys.
//
// If the map is missing a non-required key,
// it logs an INFO message unless the given logging client is nil.
func ParseConsulConfig(lc logger.LoggingClient, configMap map[string]string) (ConsulConfig, error) {
	cfg := NewConsulConfig()
	settings := &cfg.ApplicationSettings

	type confItem struct {
		target   interface{} // pointer to the variable to set
		required bool        // if true, return an error if not in the map
	}

	used := make(map[string]bool, len(configMap))
	for key, ci := range map[string]confItem{
		"AdjustLastReadOnByOrigin":     {target: &settings.AdjustLastReadOnByOrigin},
		"DepartedThresholdSeconds":     {target: &settings.DepartedThresholdSeconds},
		"DepartedCheckIntervalSeconds": {target: &settings.DepartedCheckIntervalSeconds},
		"AgeOutHours":                  {target: &settings.AgeOutHours},
		"MobilityProfileThreshold":     {target: &settings.MobilityProfileThreshold},
		"MobilityProfileHoldoffMillis": {target: &settings.MobilityProfileHoldoffMillis},
		"MobilityProfileSlope":         {target: &settings.MobilityProfileSlope},
		"DeviceServiceName":            {target: &settings.DeviceServiceName},
		"DeviceServiceURL":             {target: &settings.DeviceServiceURL},
		"MetadataServiceURL":           {target: &settings.MetadataServiceURL},
	} {
		var err error

		val, ok := configMap[key]
		if !ok {
			if ci.required {
				return cfg, errors.Wrapf(ErrMissingRequiredKey, "no value for %q", key)
			}

			if lc != nil {
				lc.Info("Using default value for config item.",
					"key", key, "value", ci.target)
			}
			continue
		}

		switch target := ci.target.(type) {
		default:
			panic(fmt.Sprintf("unhandled type for config item %q: %T",
				key, ci.target))

		case *string:
			*target = val
		case *float64:
			*target, err = strconv.ParseFloat(val, 64)
		case *bool:
			*target, err = strconv.ParseBool(val)
		case *int:
			*target, err = strconv.Atoi(val)

		case *uint:
			u, perr := strconv.ParseUint(val, 10, 0)
			err = perr
			*target = uint(u)
		}

		if err != nil {
			return cfg, errors.Wrapf(err, "failed to parse config item %q, %q", key, val)
		}

		used[key] = true
	}

	if err := settings.Validate(); err != nil {
		return cfg, err
	}

	var missed []string
	for key, val := range configMap {
		if !used[key] {
			missed = append(missed, fmt.Sprintf("%q: %q", key, val))
		}
	}

	if len(missed) != 0 {
		return cfg, errors.Wrapf(ErrUnexpectedConfigItems, "unused config items: %s", missed)
	}

	return cfg, nil
}
