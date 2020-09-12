/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package routes

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"golang.org/x/net/context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var (
	lc logger.LoggingClient
)

func TestMain(m *testing.M) {
	lc = logger.NewClient("test", false, "", "DEBUG")

	os.Exit(m.Run())
}

func TestIndex(t *testing.T) {

	request, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("Unable to create new HTTP request %v", err)
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
		t.Fatalf("Unable to create new HTTP request %v", err)
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
	eventCh := make(chan inventory.Event, 1)
	tagPro := inventory.NewTagProcessor(lc, eventCh)

	request, err := http.NewRequest("GET", "/inventory/snapshot", nil)
	if err != nil {
		t.Fatalf("Unable to create new HTTP request %v", err)
	}
	recorder := httptest.NewRecorder()
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		RawInventory(lc, writer, request, tagPro)
	})
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected no error")
	}
}
