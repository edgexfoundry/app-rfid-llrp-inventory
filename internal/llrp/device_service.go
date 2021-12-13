//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos/responses"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
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

	enableImpinjCmd = "ImpinjCustomExtensionMessage"

	capReadingName = "ReaderCapabilities"

	maxTries      = 5
	sleepInterval = 250 * time.Millisecond
)

// DSClient is a client to interact with the LLRP Device Service.
type DSClient struct {
	lc        logger.LoggingClient
	cmdClient interfaces.CommandClient
}

// NewDSClient returns a DSClient reachable at the given host URL,
// using the given http Client, which of course may be the default.
// TODO: Use new device service Clients from go-mod-core-clients when upgrading to V2 (Ireland)
//		 https://github.com/edgexfoundry/app-rfid-llrp-inventory/pull/18#discussion_r592768121
func NewDSClient(cmdClient interfaces.CommandClient, lc logger.LoggingClient) DSClient {

	return DSClient{
		cmdClient: cmdClient,
		lc:        lc,
	}
}

// GetDevices return a list of device names known to the EdgeX Metadata service.
func GetDevices(client interfaces.DeviceClient, dsName string) ([]dtos.Device, error) {
	response, err := client.DevicesByServiceName(context.Background(), dsName, 0, -1)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get device list")
	}

	return response.Devices, nil
}

// NewReader returns a TagReader instance for the given device name
// by querying the LLRP Device Service for details about it.
//
// If the Device Service isn't tracking a device with the given name,
// then this returns an error.
func (ds DSClient) NewReader(device string) (TagReader, error) {
	devCap, err := ds.GetCapabilities(device)
	if err != nil {
		return nil, err
	}

	if devCap.GeneralDeviceCapabilities == nil {
		return nil, errors.Errorf("missing general capabilities for %q", device)
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

// GetCapabilities queries the device service for a device's capabilities.
func (ds DSClient) GetCapabilities(device string) (*GetReaderCapabilitiesResponse, error) {
	var err error
	var resp *responses.EventResponse
	try := 1

	ds.lc.Debugf("Sending GET command '%s' to device '%s'", capDevCmd, device)

	// need to retry because when an offline reader goes online, and is marked Enabled, it may retuurn an error when querying capabilities because it is not 100% ready
	for try < maxTries {
		resp, err = ds.cmdClient.IssueGetCommandByName(context.Background(), device, capDevCmd, "no", "yes")
		if err != nil {
			try++
			time.Sleep(sleepInterval * time.Duration(try))
			continue
		}
		break
	}

	if err != nil {
		return nil, errors.Wrap(err, "device info request failed")
	}

	caps := &GetReaderCapabilitiesResponse{}
	for _, reading := range resp.Event.Readings {
		if reading.ResourceName == capReadingName {
			// Object value types come in as a map[string]interface{} which need to be marshalled from this rather than JSON
			err := mapstructure.Decode(reading.ObjectValue, &caps)
			if err != nil {
				return nil, errors.Wrap(err, "Unmarshal failed for reader capabilities")
			}
			break
		}
	}

	if caps == nil {
		return nil, errors.New("failed to get reader capabilities")
	}

	return caps, nil
}

// SetConfig requests the device service set a particular device's configuration.
func (ds DSClient) SetConfig(device string, conf *SetReaderConfig) error {
	confData, err := json.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "failed to marshal SetReaderConfig message")
	}

	var configMapData map[string]interface{}
	json.Unmarshal(confData, &configMapData)
	commandData := map[string]interface{}{
		"ReaderConfig": configMapData,
	}

	ds.lc.Debugf("Sending SET command '%s' to device '%s' with data '%v'", configDevCmd, device, commandData)

	_, err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(), device, configDevCmd, commandData)
	if err != nil {
		return errors.WithMessage(err, "failed to set reader config")
	}
	return nil
}

// AddROSpec adds an ROSpec on the given device.
func (ds DSClient) AddROSpec(device string, spec *ROSpec) error {
	roData, err := json.Marshal(spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal ROSpec")
	}

	var roMapData map[string]interface{}
	json.Unmarshal(roData, &roMapData)
	commandData := map[string]interface{}{
		"ROSpec": roMapData,
	}

	var data map[string]interface{}
	json.Unmarshal(roData, &data)

	ds.lc.Debugf("Sending SET command '%s' to device '%s' with data '%v'", addCmd, device, commandData)

	_, err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(), device, addCmd, commandData)
	if err != nil {
		return errors.WithMessage(err, "failed to add ROSpec")
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

// modifyROSpecState requests the device service set the given device's
// ROSpec to a particular state.
func (ds DSClient) modifyROSpecState(roCmd, device string, id uint32) error {
	data := make(map[string]string)

	data["ROSpecID"] = strconv.FormatUint(uint64(id), 10)

	ds.lc.Debugf("Sending SET command '%s' to device '%s' with data '%v'", roCmd, device, data)

	_, err := ds.cmdClient.IssueSetCommandByName(context.Background(), device, roCmd, data)
	if err != nil {
		return errors.WithMessage(err, "failed to "+roCmd)
	}

	return nil
}

// EnableCustomExt enables custom Impinj extensions.
// Note that the device in question must be registered
// with a device profile that has an enableImpinjExt deviceCommand.
func (d *ImpinjDevice) EnableCustomExt(device string, ds DSClient) error {
	data := make(map[string]string)

	data["ImpinjCustomExtensionMessage"] = "AAAAAA=="

	ds.lc.Debugf("Sending SET command '%s' to device '%s' with data '%v'", enableImpinjCmd, device, data)

	_, err := ds.cmdClient.IssueSetCommandByName(context.Background(), device, enableImpinjCmd, data)
	if err != nil {
		return errors.WithMessage(err, "failed to enable Impinj extensions")
	}

	return nil
}
