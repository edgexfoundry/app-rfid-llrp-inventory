//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"encoding/json"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

const (
	// LLRPDeviceProfile specifies the name of the device profile
	// in use for LLRP readers, used to determine device type
	LLRPDeviceProfile = "Device.LLRP.Profile"
)

func getDeviceList(respBody []byte) (deviceList []string, err error) {
	var deviceSlice []contract.Device

	err = json.Unmarshal(respBody, &deviceSlice)
	if err != nil {
		return nil, err
	}

	for _, d := range deviceSlice {

		// filter only llrp readers
		if d.Profile.Name == LLRPDeviceProfile {
			deviceList = append(deviceList, d.Name)
		}
	}
	return deviceList, nil
}

// GetDevices GET rest call to edgex-core-command to get the devices/readers list
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
		return nil, errors.Errorf("GET call to edgex-core-command to get the readers failed: %d", resp.StatusCode)
	} else {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		deviceList, err := getDeviceList(respBody)
		if err != nil {
			return nil, errors.Errorf("Unable to parse device list from EdgeX: %s", err.Error())
		}
		if len(deviceList) == 0 {
			return nil, errors.Errorf("No devices registered")
		}
		return deviceList, nil
	}
}

// SendHTTPGETRequest sends GET Request to Edgex Core Command
func SendHTTPGETRequest(endpoint string, logger logger.LoggingClient, client *http.Client) error {
	logger.Debug(http.MethodGet + " " + endpoint)
	// create New GET request
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Check & report for any error from EdgeX Core
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("GET to EdgeX Core failed with status %d; body: %q", resp.StatusCode, string(body))
	}

	logger.Debug("Response from Edgex Core: " + string(body))
	return nil

}
