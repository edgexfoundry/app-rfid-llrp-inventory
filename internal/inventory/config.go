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
	// ErrParsingConfigValue is returned when we are unable to parse the value for a config key
	ErrParsingConfigValue = errors.New("unable to parse config value for key")
	// ErrMissingRequiredKey is returned when we are unable to parse the value for a config key
	ErrMissingRequiredKey = errors.New("missing required key")
)

// Configurator is a helper type that parses EdgeX configuration to Go struct
type Configurator struct {
	lc       logger.LoggingClient
	cloneMap map[string]string
	// defaultAppSettings holds default values for each configurable item in case
	// they are not present in the configuration
	defaultAppSettings map[string]string
}

// NewConfigurator creates a new instance of a Configurator
func NewConfigurator(lc logger.LoggingClient) *Configurator {
	return &Configurator{
		lc: lc,
		defaultAppSettings: map[string]string{
			"AdjustLastReadOnByOrigin":     "true",
			"DepartedThresholdSeconds":     "600",
			"DepartedCheckIntervalSeconds": "30",
			"AgeOutHours":                  "336",

			"MobilityProfileThreshold":     "6",
			"MobilityProfileHoldoffMillis": "500",
			"MobilityProfileSlope":         "-0.008",

			"DeviceServiceName":  "edgex-device-llrp",
			"DeviceServiceURL":   "http://edgex-device-llrp:51992/",
			"MetadataServiceURL": "http://edgex-core-metadata:48081/",
		},
	}
}

// Parse takes a string map from EdgeX of the ApplicationSettings and creates a new ConsulConfig
// instance pre-filled with the values from EdgeX, as well as filling in any missing values with
// their associated defaults.
//
// It returns an error wrapping ErrUnexpectedConfigItems if there are additional unused keys and
// values after parsing is complete. This error can be safely ignored using:
// `!errors.Is(err, ErrUnexpectedConfigItems)`
// It may also return an error wrapping ErrParsingConfigValue or ErrMissingRequiredKey.
func (cr *Configurator) Parse(appSettings map[string]string) (ConsulConfig, error) {
	cfg := ConsulConfig{
		Writable: WriteableConfig{
			LogLevel: "INFO",
		},
		Aliases: make(map[string]string),
	}
	err := cr.loadAppSettings(appSettings, &cfg.ApplicationSettings)
	return cfg, err
}

// loadAppSettings is the internal function that takes the incoming strings map and parses it to fill
// in the values of the ApplicationSettings pointer.
func (cr *Configurator) loadAppSettings(configMap map[string]string, settings *ApplicationSettings) error {
	cr.cloneMap = make(map[string]string, len(configMap))
	for k, v := range configMap {
		cr.cloneMap[k] = v
	}

	var err error

	settings.AdjustLastReadOnByOrigin, err = cr.popBool("AdjustLastReadOnByOrigin")
	if err != nil {
		return wrapParseError(err, "AdjustLastReadOnByOrigin")
	}

	settings.DepartedThresholdSeconds, err = cr.popInt("DepartedThresholdSeconds")
	if err != nil {
		return wrapParseError(err, "DepartedThresholdSeconds")
	}

	settings.DepartedCheckIntervalSeconds, err = cr.popInt("DepartedCheckIntervalSeconds")
	if err != nil {
		return wrapParseError(err, "DepartedCheckIntervalSeconds")
	}

	settings.AgeOutHours, err = cr.popInt("AgeOutHours")
	if err != nil {
		return wrapParseError(err, "AgeOutHours")
	}

	settings.MobilityProfileThreshold, err = cr.popFloat64("MobilityProfileThreshold")
	if err != nil {
		return wrapParseError(err, "MobilityProfileThreshold")
	}

	settings.MobilityProfileHoldoffMillis, err = cr.popFloat64("MobilityProfileHoldoffMillis")
	if err != nil {
		return wrapParseError(err, "MobilityProfileHoldoffMillis")
	}

	settings.MobilityProfileSlope, err = cr.popFloat64("MobilityProfileSlope")
	if err != nil {
		return wrapParseError(err, "MobilityProfileSlope")
	}

	settings.DeviceServiceName, err = cr.pop("DeviceServiceName")
	if err != nil {
		return wrapParseError(err, "DeviceServiceName")
	}

	settings.DeviceServiceURL, err = cr.pop("DeviceServiceURL")
	if err != nil {
		return wrapParseError(err, "DeviceServiceURL")
	}

	settings.MetadataServiceURL, err = cr.pop("MetadataServiceURL")
	if err != nil {
		return wrapParseError(err, "MetadataServiceURL")
	}

	// in this case there were extra fields that are not in our config map.
	// these could either be outdated config options or typos
	if len(cr.cloneMap) > 0 {
		cr.lc.Warn("Got unexpected config keys and values.",
			"unexpected", fmt.Sprintf("%+v", cr.cloneMap))
		return errors.Wrapf(ErrUnexpectedConfigItems, "config map: %+v", cr.cloneMap)
	}
	return nil
}

// wrapParseError is a utility function to wrap an error parsing specified key
// with ErrParsingConfigValue
func wrapParseError(err error, key string) error {
	return errors.Wrap(err, errors.Wrap(ErrParsingConfigValue, key).Error())
}

// pop retrieves the value stored in `cloneMap` for the specified key if it exists
// and deletes it from the map. If it does not exist, it uses the default value configured
// for that key. It will return an error wrapping `ErrMissingRequiredKey` if the key is
// missing from `cloneMap` and there is no default value specified for that key.
func (cr *Configurator) pop(key string) (string, error) {
	val, ok := cr.cloneMap[key]
	if !ok {
		val, ok = cr.defaultAppSettings[key]
		if !ok {
			return "", errors.Wrap(ErrMissingRequiredKey, key)
		}
		cr.lc.Info(fmt.Sprintf("Config is missing property '%s', value has been set to the default value of '%s'.", key, val))
	}
	// delete each handled field to know if there are any un-handled ones left
	delete(cr.cloneMap, key)
	return val, nil
}

// popInt functions the same way as pop, except it will attempt to convert the value to an int
func (cr *Configurator) popInt(key string) (int, error) {
	val, err := cr.pop(key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(val)
}

// popFloat64 functions the same way as pop, except it will attempt to convert the value to a float64
func (cr *Configurator) popFloat64(key string) (float64, error) {
	val, err := cr.pop(key)
	if err != nil {
		return 0.0, err
	}
	return strconv.ParseFloat(val, 64)
}

// popBool functions the same way as pop, except it will attempt to convert the value to a bool
func (cr *Configurator) popBool(key string) (bool, error) {
	val, err := cr.pop(key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(val)
}
