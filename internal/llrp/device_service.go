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
	"github.com/pkg/errors"
)

// These are the names of deviceResource and deviceCommands
// this expects the devices are using in their registered deviceProfiles.
const (
	basePath = "/api/v2/device/name/" // paths are {base/}{device}{/target}

	capDevCmd    = "ReaderCapabilities"
	configDevCmd = "ReaderConfig"
	addCmd       = "ROSpec"
	enableCmd    = "enableROSpec"
	disableCmd   = "disableROSpec"
	stopCmd      = "stopROSpec"
	startCmd     = "startROSpec"
	deleteCmd    = "deleteROSpec"

	enableImpinjCmd = "/enableImpinjExt"

	capReadingName = "ReaderCapabilities"

	maxBody = 100 * 1024

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
	/*r, err := ds.tryGet(device + capDevCmd)
	if err != nil {
		return nil, errors.Wrapf(err, "device info request failed for device %s", device)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, errors.Errorf("device info request failed with status %d", r.StatusCode)
	}*/

	resp, err := ds.cmdClient.IssueGetCommandByName(context.Background(), device, capDevCmd, "false", "true")
	if err != nil {
		return nil, errors.Wrap(err, "device info request failed")
	}

	// content, err := ioutil.ReadAll(io.LimitReader(r.Body, maxBody))
	// if err != nil {
	// 	return nil, errors.Wrap(err, "device info request failed")
	// }

	var caps *GetReaderCapabilitiesResponse
	for _, reading := range resp.Event.Readings {
		if reading.ResourceName == capReadingName {
			err := ds.unMarshallObjectValue(reading.ObjectValue, caps, "device info request failed")
			if err != nil {
				return nil, err
			}
			break
		}
	}

	if caps == nil {
		return nil, errors.New("failed to get reader capabilities")
	}

	return caps, nil
}

func (ds DSClient) unMarshallObjectValue(objectValue interface{}, target interface{}, errMessage string) error {
	data, err := json.Marshal(objectValue)
	if err != nil {
		return errors.Wrap(err, errMessage)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return errors.Wrap(err, errMessage)
	}
	return nil
}

// SetConfig requests the device service set a particular device's configuration.
func (ds DSClient) SetConfig(device string, conf *SetReaderConfig) error {
	confData, err := json.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "failed to marshal SetReaderConfig message")
	}

	var data map[string]interface{}
	json.Unmarshal(confData, &data)

	_, err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(), device, configDevCmd, data)
	if err != nil{
		return errors.WithMessage(err,"failed to set reader config") //check return with lenny
	}
	return nil
}

// AddROSpec adds an ROSpec on the given device.
//check work with lenny
func (ds DSClient) AddROSpec(device string, spec *ROSpec) error {
	roData, err := json.Marshal(spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal ROSpec")
	}

	var data map[string]interface{}
	json.Unmarshal(roData,&data)

	_, err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(),device,addCmd,data)
	if err != nil{
		return errors.WithMessage(err, "failed to add ROSpec")
	}
	return nil
	// edgexReq, err := json.Marshal(struct{ ROSpec string }{string(roData)})
	// if err != nil {
	// 	return errors.Wrap(err, "failed to marshal ReaderConfig edgex request")
	// }

	// return errors.WithMessage(ds.put(device+addCmd, edgexReq),
	// 	"failed to add ROSpec")
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

//Confirm with lenny
func (ds DSClient) modifyROSpecState(roCmd, device string, id uint32) error {
	modData, err := json.Marshal(struct{ ROSpecID string }{strconv.FormatUint(uint64(id), 10)})
	if err != nil{
		return errors.Wrap(err, "failed to marshal ROSpec")
	}

	var data map[string]interface{}
	json.Unmarshal(modData,&data)

	_,err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(),device,roCmd,data)
	if err != nil{
		return errors.WithMessage(err, "failed to "+roCmd[1:]) //check with leny
	}
	// edgexReq, err := json.Marshal(struct{ ROSpecID string }{strconv.FormatUint(uint64(id), 10)})
	// if err != nil {
	// 	return errors.Wrap(err, "failed to marshal ROSpec")
	// }

	// this uses roCmd[1:] because it starts with "/"
	// return errors.WithMessage(ds.put(device+roCmd, edgexReq),
	// 	"failed to "+roCmd[1:])
	return nil
}

// EnableCustomExt enables custom Impinj extensions.
// Note that the device in question must be registered
// with a device profile that has an enableImpinjExt deviceCommand.

//confirm with lenny
func (d *ImpinjDevice) EnableCustomExt(name string, ds DSClient) error {
	customExtData,err:= json.Marshal(struct{ImpinjCustomExtensionMessage string}{"AAAAAA=="})
	if err != nil{
		return errors.Wrap(err, "faile to marshal ImpinjCustomExtensionMessage")
	}
	var data map[string]interface{}
	json.Unmarshal(customExtData, &data)

	_,err = ds.cmdClient.IssueSetCommandByNameWithObject(context.Background(),name,enableImpinjCmd,data)
	if err != nil {
		return errors.WithMessage(err,"failed to enable Impinj extensions")
	}

	// enableExtension := ds.put(name+enableImpinjCmd,
	// 	[]byte(`{"ImpinjCustomExtensionMessage":"AAAAAA=="}`))
	// msg := "failed to enable Impinj extensions"
	// return errors.WithMessage(enableExtension, msg)
	return nil
}

// tryGet attempts to make an HTTP GET call to the device service at the specified path. It will
// try up to maxTries times and sleeps sleepInterval*tryCount in-between tries. It will return the
// raw response and error objects of the final HTTP call that was completed.
// It determines a successful try as anything returning 2xx http code.
// Note: if err is nil, caller is expected to close response body
// func (ds DSClient) tryGet(path string) (resp *http.Response, err error) {
// 	req, err := http.NewRequest(http.MethodGet, ds.baseURL+path, nil)
// 	if err != nil {
// 		return nil, errors.Wrapf(err, "error creating new http GET request for path %s", path)
// 	}

// 	// start at 1 since this is our first try
// 	for i := 1; ; i++ {
// 		resp, err = ds.httpClient.Do(req)
// 		if err != nil {
// 			ds.lc.Error(fmt.Sprintf("device service HTTP GET %s attempt returned an error: %v", path, err))
// 			if i < maxTries { // if we have tries left, sleep and retry
// 				time.Sleep(sleepInterval * time.Duration(i))
// 				continue
// 			}
// 		} else if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
// 			ds.lc.Error(fmt.Sprintf("device service HTTP GET %s attempt returned http error code %d", path, resp.StatusCode))
// 			if i < maxTries { // if we have tries left, sleep and retry
// 				_ = resp.Body.Close() // close the body so that the request may be re-used
// 				time.Sleep(sleepInterval * time.Duration(i))
// 				continue
// 			}
// 		}
// 		// if we reached here that means either the request was successful, or
// 		// we have no more tries left, so exit loop
// 		break
// 	}

// 	return resp, err
// }
