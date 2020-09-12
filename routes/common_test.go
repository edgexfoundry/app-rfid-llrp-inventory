/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package routes

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"golang.org/x/net/context"
	"net/http"
	"testing"
)

func TestGetSettingsHandler(t *testing.T) {

	var testLogger logger.LoggingClient
	var testAppSettings map[string]string
	settings := SettingsHandler{Logger: testLogger, AppSettings: testAppSettings}

	request, err := http.NewRequest("GET", "/command/readers", nil)
	if err != nil {
		t.Fatalf("Unable to create new HTTP request %v", err)
	}
	// Invalid request
	_, _, err = GetSettingsHandler(request)
	if err == nil {
		t.Error("Expected error as no context has been set for the request")
	}

	// Invalid context set for the request
	ctx := context.WithValue(request.Context(), "InvalidKey", settings)
	_, _, err = GetSettingsHandler(request.WithContext(ctx))
	if err == nil {
		t.Error("Expected error as the request context is invalid")
	}

	// Nil logger and appSettings
	ctx = context.WithValue(request.Context(), SettingsKey, settings)
	log, appSettings, err := GetSettingsHandler(request.WithContext(ctx))
	if appSettings != nil || log != nil {
		t.Errorf("Expected logger and appSettings to be nil")
	}
	if err == nil {
		t.Errorf("Expected error because logger/appSettings should be nil")
	}

	// Valid request
	testAppSettings = make(map[string]string)
	testAppSettings["test"] = "testConfigValue"
	testLogger = logger.NewClient("test", false, "", "DEBUG")
	settings = SettingsHandler{Logger: testLogger, AppSettings: testAppSettings}

	ctx = context.WithValue(request.Context(), SettingsKey, settings)
	log, appSettings, err = GetSettingsHandler(request.WithContext(ctx))
	if appSettings == nil || log == nil {
		t.Errorf("Expected appSettings and logger not to be nil")
	}
	if err != nil {
		t.Errorf("Expected no error: %s", err.Error())
	}
}

func TestGetAppSetting(t *testing.T) {

	expectedAppSetting := "testConfigValue"
	testSettings := make(map[string]string)
	testSettings["correctConfigKey"] = expectedAppSetting

	actualAppSetting, err := GetAppSetting(testSettings, "correctConfigKey")
	if err != nil {
		t.Errorf("No error was expected: %s", err.Error())
	}
	if actualAppSetting != expectedAppSetting {
		t.Error("Expected and actual app settings do not match")
	}

	actualAppSetting, err = GetAppSetting(testSettings, "wrongKey")
	if err == nil || actualAppSetting != "" {
		t.Error("Error was expected as wrong config parameter was passed")
	}
}

func TestGetDeviceList(t *testing.T) {

	sampleResponseBody := []byte(`[{
	"id": "8bb0d27a",
	"name": "LLRPDeviceService",
	"adminState": "UNLOCKED",
	"operatingState": "ENABLED",
	"protocols": {
		"tcp": {
			"host": "edgex-device-llrp",
			"port": "8789"
		}
	},
	"service": {
		"id": "d3da9",
		"name": "edgex-device-llrp"
	},
	"profile": {
		"id": "936ef2",
		"name": "Device.LLRP.Profile"
	}
	}]`)

	deviceList, err := GetDeviceList(sampleResponseBody)
	if err != nil {
		t.Errorf("Expected error to be nil:%s", err.Error())
	}
	if len(deviceList) == 0 {
		t.Error("Expected device list not to be empty")
	}

	invalidResponseBody := []byte(`test`)
	deviceList, err = GetDeviceList(invalidResponseBody)
	if err == nil {
		t.Error("Expected error")
	}
	if deviceList != nil {
		t.Error("Expected device list to be nil")
	}
}
