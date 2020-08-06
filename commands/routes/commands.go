//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"encoding/json"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	httpTimeout = 60 * time.Second
)

var (
	client = &http.Client{
		Timeout: httpTimeout,
	}
)

// Index returns main page
func Index(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "res/html/index.html")
}

// RawInventory returns a handler bound to the TagProcessor.
// When called, it returns the raw inventory algorithm data.
func RawInventory(lc logger.LoggingClient, tagPro *inventory.TagProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		bytes, err := json.Marshal(tagPro.GetRawInventory())

		if err != nil {
			lc.Error("Failed to marshal inventory", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err = w.Write(bytes); err != nil {
			lc.Error("Failed to write inventory response.", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// StartReaders sends start/stop reading command via EdgeX Core Command API
func StartReaders(lc logger.LoggingClient, apiBase, cmdName, deviceURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {

		deviceList, err := GetDevices(deviceURL, client)
		if err != nil {
			lc.Error("Failed to get devices", "error", err.Error())

			w.WriteHeader(http.StatusInternalServerError)
			if _, err := w.Write([]byte("Unable to complete request.")); err != nil {
				lc.Error("Failed to write response.", "error", err.Error())
			}
			return
		}

		if len(deviceList) == 0 {
			lc.Info("No devices.")
			return
		}

		for _, deviceName := range deviceList {
			endpoint := apiBase + "/" + deviceName + "/command/" + cmdName
			go func(endpoint string) {
				req, err := http.NewRequest(http.MethodGet, endpoint, nil)
				if err != nil {
					lc.Error("Failed to construct request.", "error", err)
					return
				}

				resp, err := client.Do(req)
				if err != nil {
					lc.Error("Request failed.", "command", cmdName, "error", err)
					return
				}
				defer resp.Body.Close()

				if 200 <= resp.StatusCode && resp.StatusCode < 300 {
					// Best effort: see if the body has anything useful.
					body, _ := ioutil.ReadAll(resp.Body)
					if len(body) > 0 {
						lc.Error("Request failed.", "command", cmdName,
							"endpoint", endpoint, "status", resp.StatusCode,
							"body", string(body))
					} else {
						lc.Error("Request failed.", "command", cmdName,
							"endpoint", endpoint, "status", resp.StatusCode)
					}
					return
				}
			}(endpoint)
		}

		w.WriteHeader(http.StatusAccepted)
		if _, err := w.Write([]byte("Request received.")); err != nil {
			lc.Error("Failed to write response.", "error", err.Error())
		}
	}
}

// SetBehaviors sends command to set/apply behavior command
func SetBehaviors() http.HandlerFunc {
	return func(writer http.ResponseWriter, req *http.Request) {
		// TODO
	}
}
