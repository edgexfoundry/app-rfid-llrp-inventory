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

type WriteableConfig struct {
	LogLevel string
}

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
			DeviceServiceURL:             "http://edgex-device-llrp:51992/",
			MetadataServiceURL:           "http://edgex-core-metadata:48081/",
			DepartedThresholdSeconds:     600,
			DepartedCheckIntervalSeconds: 30,
			AgeOutHours:                  336,
			AdjustLastReadOnByOrigin:     true,
		},
	}
}

// confItem is used when parsing a config map.
type confItem struct {
	target   interface{}  // target is a pointer to the variable to set
	check    func() error // check is an optional validation function, called after setting target.
	required bool         // required indicates whether the config item is required or optional.
}

// uintBounds parameterizes bounds checks on uint values.
type uintBounds struct {
	min, max       uint
	chkMin, chkMax bool
}

// validate returns a validation function bound to the given pointer.
// When the returned function is called, it returns nil
// if the pointer's value is within the uintBounds,
// or an error indicating the correct bounds for the variable.
func (ub uintBounds) validate(u *uint) func() error {
	if u == nil {
		panic("missing pointer to value to check")
	}

	return func() error {
		valid := true
		if ub.chkMin {
			valid = valid && *u >= ub.min
		}

		if ub.chkMax {
			valid = valid && *u <= ub.max
		}

		if valid {
			return nil
		}

		msg := "value is %d, but must be "
		switch {
		case ub.chkMin && ub.chkMax:
			return errors.Wrapf(ErrOutOfRange, msg+">= %d and <= %d", *u, ub.min, ub.max)
		case ub.chkMin:
			return errors.Wrapf(ErrOutOfRange, msg+">= %d", *u, ub.min)
		default:
			return errors.Wrapf(ErrOutOfRange, msg+"<= %d", *u, ub.max)
		}
	}
}

// opt returns an optional confItem for target u and validation from these bounds.
func (ub uintBounds) opt(u *uint) confItem {
	return confItem{
		target: u,
		check:  ub.validate(u),
	}
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

	// gtZero is used to generate config items for optional uints that must be >0.
	gtZero := uintBounds{chkMin: true, min: 1}

	used := make(map[string]bool, len(configMap))
	for key, ci := range map[string]confItem{
		"AdjustLastReadOnByOrigin":     {target: &settings.AdjustLastReadOnByOrigin},
		"DepartedThresholdSeconds":     gtZero.opt(&settings.DepartedThresholdSeconds),
		"DepartedCheckIntervalSeconds": gtZero.opt(&settings.DepartedCheckIntervalSeconds),
		"AgeOutHours":                  gtZero.opt(&settings.AgeOutHours),
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

		if ci.check != nil {
			if err := ci.check(); err != nil {
				return cfg, errors.WithMessagef(err, "invalid config for %q, %q", key, val)
			}
		}

		used[key] = true
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
