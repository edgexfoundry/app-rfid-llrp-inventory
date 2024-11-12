//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
)

// These are the names of deviceResource and deviceCommands
// this expects the devices are using in their registered deviceProfiles.
const (
	capDevCmd    = "ReaderCapabilities"
	configDevCmd = "ReaderConfig"
	addCmd       = "ROSpec"
	enableCmd    = "enableROSpec"
	disableCmd   = "disableROSpec"
	stopCmd      = "stopROSpec"
	startCmd     = "startROSpec"
	deleteCmd    = "deleteROSpec"

	enableImpinjCmd = "EnableImpinjExtensions"

	capReadingName = "ReaderCapabilities"

	maxTries      = 5
	sleepInterval = 250 * time.Millisecond
)

// DSClient is a client to interact with the LLRP Device Service.
type DSClient struct {
	lc        logger.LoggingClient
	cmdClient interfaces.CommandClient
}

// NewDSClient returns a DSClient which uses the command client to issue commands to the devices
func NewDSClient(cmdClient interfaces.CommandClient, lc logger.LoggingClient) DSClient {
	return DSClient{
		cmdClient: cmdClient,
		lc:        lc,
	}
}

// NewReader returns a TagReader instance for the given device name
// by querying the LLRP Device Service for details about it, via command client.
//
// If the Device Service isn't tracking a device with the given name,
// then this returns an error.
func (ds DSClient) NewReader(device string) (TagReader, error) {
	devCap, err := ds.GetCapabilities(device)
	if err != nil {
		return nil, err
	}

	if devCap.GeneralDeviceCapabilities == nil {
		return nil, fmt.Errorf("missing general capabilities for %q", device)
	}

	var tr TagReader
	switch VendorPEN(devCap.GeneralDeviceCapabilities.DeviceManufacturer) {
	case PENImpinj:
		impDev, err := NewImpinjDevice(devCap)
		if err != nil {
			return nil, err
		}

		if err := impDev.EnableCustomExt(device, ds); err != nil {
			return nil, err
		}

		if err := ds.SetConfig(device, impDev.NewConfig()); err != nil {
			return nil, err
		}

		tr = impDev
	default:
		basic, err := NewBasicDevice(devCap)
		if err != nil {
			return nil, err
		}

		if err := ds.SetConfig(device, basic.NewConfig()); err != nil {
			return nil, err
		}

		tr = basic
	}

	return tr, nil
}

// GetCapabilities queries for a device's capabilities.
func (ds DSClient) GetCapabilities(device string) (*GetReaderCapabilitiesResponse, error) {
	ds.lc.Debugf("Sending GET command '%s' to device '%s'", capDevCmd, device)

	resp, err := ds.cmdClient.IssueGetCommandByName(context.Background(), device, capDevCmd, false, true)
	for try := 1; err != nil && try < maxTries; try++ {
		// when an offline reader comes back online, it may return an error querying the capabilities due to a
		// slight delay when the device service updates the reader's OperatingState. So lets sleep a bit and retry.
		time.Sleep(sleepInterval * time.Duration(try))
		resp, err = ds.cmdClient.IssueGetCommandByName(context.Background(), device, capDevCmd, false, true)
	}

	if err != nil {
		return nil, fmt.Errorf("device info request failed: %w", err)
	}

	caps := &GetReaderCapabilitiesResponse{}
	for _, reading := range resp.Event.Readings {
		if reading.ResourceName == capReadingName {
			// reading objectvalue is an interface which will be marshalled into the objectvalue as a map[string]interface{}
			// in order to get this into the reader capabilities struct we need to first marshal it back to JSON
			data, err := json.Marshal(reading.ObjectValue)
			if err != nil {
				return nil, fmt.Errorf("marshal failed for reading object value (reader capabilities): %w", err)
			}
			err = json.Unmarshal(data, &caps)
			if err != nil {
				return nil, fmt.Errorf("unmarshal failed for reader capabilities: %w", err)
			}
			break
		}
	}

	if caps == nil {
		return nil, errors.New("failed to get reader capabilities")
	}

	return caps, nil
}

// SetConfig requests to set a particular device's configuration.
func (ds DSClient) SetConfig(device string, conf *SetReaderConfig) error {
	confData, err := json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("failed to marshal SetReaderConfig message: %w", err)
	}

	var configMapData map[string]interface{}
	err = json.Unmarshal(confData, &configMapData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal SetReaderConfig message: %w", err)
	}

	commandData := map[string]interface{}{
		"ReaderConfig": configMapData,
	}

	ds.lc.Debugf("Sending SET command '%s' to device '%s' with data '%v'", configDevCmd, device, commandData)

	_, err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(), device, configDevCmd, commandData)
	if err != nil {
		return fmt.Errorf("failed to set reader config: %v", err)
	}
	return nil
}

// AddROSpec adds an ROSpec on the given device.
func (ds DSClient) AddROSpec(device string, spec *ROSpec) error {
	roData, err := json.Marshal(spec)
	if err != nil {
		return fmt.Errorf("failed to marshal ROSpec: %w", err)
	}

	var roMapData map[string]interface{}
	err = json.Unmarshal(roData, &roMapData)
	if err != nil {
		return fmt.Errorf("failed to unmarshal ROSpec: %w", err)
	}

	commandData := map[string]interface{}{
		"ROSpec": roMapData,
	}

	ds.lc.Debugf("Sending SET command '%s' to device '%s' with data '%v'", addCmd, device, commandData)

	_, err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(), device, addCmd, commandData)
	if err != nil {
		return fmt.Errorf("failed to add ROSpec: %v", err)
	}
	return nil
}

// EnableROSpec enables the ROSpec with the given ID on the given device.
func (ds DSClient) EnableROSpec(device string, id uint32) error {
	return ds.modifyROSpecState(enableCmd, device, id)
}

// DisableROSpec disables the ROSpec with the given ID on the given device.
func (ds DSClient) DisableROSpec(device string, id uint32) error {
	return ds.modifyROSpecState(disableCmd, device, id)
}

// StopROSpec stops the ROSpec with the given ID on the given device.
func (ds DSClient) StopROSpec(device string, id uint32) error {
	return ds.modifyROSpecState(stopCmd, device, id)
}

// StartROSpec starts the ROSpec with the given ID on the given device.
func (ds DSClient) StartROSpec(device string, id uint32) error {
	return ds.modifyROSpecState(startCmd, device, id)
}

// DeleteROSpec deletes the ROSpec with the given ID on the given device.
func (ds DSClient) DeleteROSpec(device string, id uint32) error {
	return ds.modifyROSpecState(deleteCmd, device, id)
}

// DeleteAllROSpecs deletes all the ROSpecs on the given device.
func (ds DSClient) DeleteAllROSpecs(device string) error {
	return ds.modifyROSpecState(deleteCmd, device, 0)
}

// modifyROSpecState requests to set the given device's ROSpec to a particular state.
func (ds DSClient) modifyROSpecState(roCmd, device string, id uint32) error {
	data := make(map[string]string)

	data["ROSpecID"] = strconv.FormatUint(uint64(id), 10)

	ds.lc.Debugf("Sending SET command '%s' to device '%s' with data '%v'", roCmd, device, data)

	_, err := ds.cmdClient.IssueSetCommandByName(context.Background(), device, roCmd, data)
	if err != nil {
		return fmt.Errorf("failed to "+roCmd+": %v", err)
	}

	return nil
}

// EnableCustomExt enables custom Impinj extensions.
// Note that the device in question must be registered
// with a device profile that has an enableImpinjExt deviceCommand.
func (d *ImpinjDevice) EnableCustomExt(device string, ds DSClient) error {
	data := make(map[string]string)

	ds.lc.Debugf("Sending SET command '%s' to device '%s'", enableImpinjCmd, device)

	// the Device profile for impinj has a defaultvalue, so passing empty map
	_, err := ds.cmdClient.IssueSetCommandByName(context.Background(), device, enableImpinjCmd, data)
	if err != nil {
		return fmt.Errorf("failed to enable Impinj extensions: %v", err)
	}

	return nil
}
