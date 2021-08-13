//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// These are the names of deviceResource and deviceCommands
// this expects the devices are using in their registered deviceProfiles.
const (
	basePath = "/api/v1/device/name/" // paths are {base/}{device}{/target}

	capDevCmd    = "/capabilities"
	configDevCmd = "/config"
	addCmd       = "/roSpec"
	enableCmd    = "/enableROSpec"
	disableCmd   = "/disableROSpec"
	stopCmd      = "/stopROSpec"
	startCmd     = "/startROSpec"
	deleteCmd    = "/deleteROSpec"

	enableImpinjCmd = "/enableImpinjExt"

	capReadingName = "ReaderCapabilities"

	maxBody = 100 * 1024

	maxTries      = 5
	sleepInterval = 250 * time.Millisecond
)

// DSClient is a client to interact with the LLRP Device Service.
type DSClient struct {
	baseURL    string
	httpClient *http.Client
	lc         logger.LoggingClient
}

// NewDSClient returns a DSClient reachable at the given host URL,
// using the given http Client, which of course may be the default.
// TODO: Use new device service Clients from go-mod-core-clients when upgrading to V2 (Ireland)
//		 https://github.com/edgexfoundry/app-rfid-llrp-inventory/pull/18#discussion_r592768121
func NewDSClient(host *url.URL, c *http.Client, lc logger.LoggingClient) DSClient {
	base := url.URL{
		Scheme: host.Scheme,
		Opaque: host.Opaque,
		User:   host.User,
		Host:   host.Host,
		Path:   basePath,
	}

	return DSClient{
		baseURL:    base.String(),
		httpClient: c,
		lc:         lc,
	}
}

// GetDevices return a list of device names known to the EdgeX Metadata service.
func GetDevices(metadataDevicesURL string, client *http.Client) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, metadataDevicesURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET returned unexpected status: %d", resp.StatusCode)
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
	r, err := ds.tryGet(device + capDevCmd)
	if err != nil {
		return nil, errors.Wrapf(err, "device info request failed for device %s", device)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, errors.Errorf("device info request failed with status %d", r.StatusCode)
	}

	content, err := ioutil.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		return nil, errors.Wrap(err, "device info request failed")
	}

	type edgexResp struct {
		Readings []struct {
			Name, Value string
		}
	}

	var resp edgexResp
	if err := json.Unmarshal(content, &resp); err != nil {
		return nil, errors.Wrap(err, "device info request failed")
	}

	var caps *GetReaderCapabilitiesResponse
	for _, reading := range resp.Readings {
		if reading.Name == capReadingName {
			caps = &GetReaderCapabilitiesResponse{}
			if err := json.Unmarshal([]byte(reading.Value), caps); err != nil {
				return nil, errors.Wrap(err, "device info request failed")
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

	edgexReq, err := json.Marshal(struct{ ReaderConfig string }{string(confData)})
	if err != nil {
		return errors.Wrap(err, "failed to marshal ReaderConfig edgex request")
	}

	return errors.WithMessage(ds.put(device+configDevCmd, edgexReq),
		"failed to set reader config")
}

// AddROSpec adds an ROSpec on the given device.
func (ds DSClient) AddROSpec(device string, spec *ROSpec) error {
	roData, err := json.Marshal(spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal ROSpec")
	}

	edgexReq, err := json.Marshal(struct{ ROSpec string }{string(roData)})
	if err != nil {
		return errors.Wrap(err, "failed to marshal ReaderConfig edgex request")
	}

	return errors.WithMessage(ds.put(device+addCmd, edgexReq),
		"failed to add ROSpec")
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
	edgexReq, err := json.Marshal(struct{ ROSpecID string }{strconv.FormatUint(uint64(id), 10)})
	if err != nil {
		return errors.Wrap(err, "failed to marshal ROSpec")
	}

	// this uses roCmd[1:] because it starts with "/"
	return errors.WithMessage(ds.put(device+roCmd, edgexReq),
		"failed to "+roCmd[1:])
}

// EnableCustomExt enables custom Impinj extensions.
// Note that the device in question must be registered
// with a device profile that has an enableImpinjExt deviceCommand.
func (d *ImpinjDevice) EnableCustomExt(name string, ds DSClient) error {
	enableExtension := ds.put(name+enableImpinjCmd,
		[]byte(`{"ImpinjCustomExtensionMessage":"AAAAAA=="}`))
	msg := "failed to enable Impinj extensions"
	return errors.WithMessage(enableExtension, msg)
}

// put PUTs the data to the device service path.
func (ds DSClient) put(path string, data []byte) error {
	req, err := http.NewRequest("PUT", ds.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return errors.Wrap(err, "failed to create device service request")
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to PUT to device service")
	}

	resp.Body.Close()

	if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// tryGet attempts to make an HTTP GET call to the device service at the specified path. It will
// try up to maxTries times and sleeps sleepInterval*tryCount in-between tries. It will return the
// raw response and error objects of the final HTTP call that was completed.
// It determines a successful try as anything returning 2xx http code.
// Note: if err is nil, caller is expected to close response body
func (ds DSClient) tryGet(path string) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodGet, ds.baseURL+path, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error creating new http GET request for path %s", path)
	}

	// start at 1 since this is our first try
	for i := 1; ; i++ {
		resp, err = ds.httpClient.Do(req)
		if err != nil {
			ds.lc.Error(fmt.Sprintf("device service HTTP GET %s attempt returned an error: %v", path, err))
			if i < maxTries { // if we have tries left, sleep and retry
				time.Sleep(sleepInterval * time.Duration(i))
				continue
			}
		} else if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
			ds.lc.Error(fmt.Sprintf("device service HTTP GET %s attempt returned http error code %d", path, resp.StatusCode))
			if i < maxTries { // if we have tries left, sleep and retry
				_ = resp.Body.Close() // close the body so that the request may be re-used
				time.Sleep(sleepInterval * time.Duration(i))
				continue
			}
		}
		// if we reached here that means either the request was successful, or
		// we have no more tries left, so exit loop
		break
	}

	return resp, err
}
