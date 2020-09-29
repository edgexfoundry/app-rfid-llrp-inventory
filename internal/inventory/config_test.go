//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func testAppSettings() map[string]string {
	// NOTE: If you change this, you MUST update `TestConfigurator_Parse`!
	return map[string]string{
		"AdjustLastReadOnByOrigin":     "FALSE",
		"DepartedThresholdSeconds":     "1000",
		"DepartedCheckIntervalSeconds": "5",
		"AgeOutHours":                  "600",

		"MobilityProfileThreshold":     "5.0",
		"MobilityProfileHoldoffMillis": "1250.0",
		"MobilityProfileSlope":         "-0.0055",

		"DeviceServiceName":  "testing",
		"DeviceServiceURL":   "http://testing:51992/",
		"MetadataServiceURL": "http://testing:48081/",
	}
}

func TestConfigurator_Parse(t *testing.T) {
	settings := testAppSettings()
	cr := NewConfigurator(lc)

	consulConfig, err := cr.Parse(settings)
	if err != nil {
		t.Fatalf("got err: %s", err.Error())
	}

	c := consulConfig.ApplicationSettings
	if c.AdjustLastReadOnByOrigin != false ||
		c.DepartedThresholdSeconds != 1000 ||
		c.DepartedCheckIntervalSeconds != 5 ||
		c.AgeOutHours != 600 ||
		c.MobilityProfileThreshold != 5.0 ||
		c.MobilityProfileHoldoffMillis != 1250.0 ||
		c.MobilityProfileSlope != -0.0055 ||
		c.DeviceServiceName != "testing" ||
		c.DeviceServiceURL != "http://testing:51992/" ||
		c.MetadataServiceURL != "http://testing:48081/" {

		t.Errorf("One of the value fields is incorrect.\nOriginal: %+v\nParsed: %+v", settings, c)
	}
}

func TestEmptyConfigDefaults(t *testing.T) {
	cr := NewConfigurator(lc)
	consolConfig, err := cr.Parse(map[string]string{})
	if err != nil {
		t.Fatalf("got err: %s", err.Error())
	}

	configValue := reflect.ValueOf(&consolConfig.ApplicationSettings).Elem()
	for i := 0; i < configValue.NumField(); i++ {
		typeField := configValue.Type().Field(i)
		valueField := configValue.Field(i)
		valueStr := fmt.Sprintf("%v", valueField.Interface())
		valueStr = strings.ReplaceAll(valueStr, "[", "")
		valueStr = strings.ReplaceAll(valueStr, "]", "")
		if valueStr != cr.defaultAppSettings[typeField.Name] {
			t.Errorf("Field %s, expected value %q, but got %q",
				typeField.Name, cr.defaultAppSettings[typeField.Name], valueStr)
		}
	}
}

func TestErrUnexpectedConfigItems(t *testing.T) {
	cfg := map[string]string{
		"foo": "bar",
	}
	cr := NewConfigurator(lc)
	if _, err := cr.Parse(cfg); !errors.Is(err, ErrUnexpectedConfigItems) {
		t.Fatalf("expected ErrUnexpectedConfigItems, but got: %v", err)
	}
}
