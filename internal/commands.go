//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package internal

import (
	"encoding/json"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
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

// RawInventory returns a handler bound to the TagProcessor.
// When called, it returns the raw inventory algorithm data.
func RawInventory(lc logger.LoggingClient, tagPro *inventory.TagProcessor) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		payload, err := json.Marshal(tagPro.GetRawInventory())
		if err != nil {
			lc.Error("Failed to marshal inventory", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err = w.Write(payload); err != nil {
			lc.Error("Failed to write inventory response.", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// GetDevices return a list of device names.
func GetDevices(devicesURL string, client *http.Client) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, devicesURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET failed with status: %d", resp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ds := &[]struct{ Name string }{}
	if err := json.Unmarshal(respBody, ds); err != nil {
		return nil, errors.Wrap(err, "failed to parse EdgeX device list")
	}

	deviceList := make([]string, len(*ds))
	for i, dev := range *ds {
		deviceList[i] = dev.Name
	}
	return deviceList, nil
}
