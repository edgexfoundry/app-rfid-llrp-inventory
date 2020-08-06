/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package routes

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendHTTPGETRequest(t *testing.T) {

	// Valid request
	testLogger := logger.NewClient("test", false, "", "DEBUG")
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/startReading" {
			t.Errorf("Expected request to be '/startReading', received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}
	}))

	defer testServer.Close()

	err := SendHTTPGETRequest(testServer.URL+"/startReading", testLogger, NewHTTPClient())
	if err != nil {
		t.Errorf("Expected no error:%s", err)
	}

	// Bad request
	testServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	}))

	defer testServer.Close()

	err = SendHTTPGETRequest(testServer.URL+"/startReading", testLogger, NewHTTPClient())
	if err == nil {
		t.Errorf("Expected error because of bad request")
	}
}

func TestSendHTTPGetDevicesRequest(t *testing.T) {

	// wrong app settings
	testAppSettings := map[string]string{"wrongKey": "testValue"}
	_, err := SendHTTPGetDevicesRequest(testAppSettings, NewHTTPClient())
	if err == nil {
		t.Errorf("Expected error because of invalid appSettings")
	}

	// Bad request with valid app settings
	testServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
	}))

	defer testServer.Close()

	testAppSettings[GetDevicesApi] = testServer.URL + "/testGetDevices"
	_, err = SendHTTPGetDevicesRequest(testAppSettings, NewHTTPClient())
	if err == nil {
		t.Errorf("Expected error because of bad request")
	}

	// valid request
	testServer = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.EscapedPath() != "/testGetDevices" {
			t.Errorf("Expected request does not match the received %s",
				request.URL.EscapedPath())
		}
		if request.Method != "GET" {
			t.Errorf("Expected 'GET' request, received '%s", request.Method)
		}

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

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(sampleResponseBody)
	}))

	defer testServer.Close()

	testAppSettings[GetDevicesApi] = testServer.URL + "/testGetDevices"
	deviceList, err := SendHTTPGetDevicesRequest(testAppSettings, NewHTTPClient())
	if err != nil {
		t.Errorf("Expected no error:%s", err)
	}
	if len(deviceList) == 0 {
		t.Errorf("Expected device list not to be nil")
	}
}
