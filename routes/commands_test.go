/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package routes

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/inventory"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIndex(t *testing.T) {

	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(Index)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected error as the request has no context set")
	}
}

func TestPing(t *testing.T) {

	request, err := http.NewRequest("GET", "/ping", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(Ping)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected error as the request has no context set")
	}

	// Valid request
	testAppSettings := make(map[string]string)
	testAppSettings["test"] = "testConfigValue"
	testLogger := logger.NewClient("test", false, "", "DEBUG")
	settings := SettingsHandler{Logger: testLogger, AppSettings: testAppSettings}

	recorder = httptest.NewRecorder()
	ctx := context.WithValue(request.Context(), SettingsKey, settings)
	handler.ServeHTTP(recorder, request.WithContext(ctx))

	expectedResponse := "pong"
	if recorder.Body.String() != expectedResponse {
		t.Errorf("Expected response does not match the actual response: %s", recorder.Body.String())
	}
}

func TestRawInventory(t *testing.T) {

	request, err := http.NewRequest("GET", "/inventory/raw", nil)
	if err != nil {
		t.Errorf("Unable to create new HTTP request %s", err.Error())
	}
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(RawInventory)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected error as the request has no context set")
	}

	// Valid request
	testAppSettings := make(map[string]string)
	testAppSettings["test"] = "testConfigValue"
	testLogger := logger.NewClient("test", false, "", "DEBUG")
	settings := SettingsHandler{Logger: testLogger, AppSettings: testAppSettings}
	inventory.NewTagProcessor(testLogger)

	recorder = httptest.NewRecorder()
	ctx := context.WithValue(request.Context(), SettingsKey, settings)
	handler.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK {
		t.Errorf("Expected no error")
	}
}
