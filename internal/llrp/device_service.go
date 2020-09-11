//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"bytes"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

type DSClient struct {
	baseURL    string
	httpClient *http.Client
}

type TProc interface {
	NewROSpec(b Behavior, e Environment) (*ROSpec, error)
	FillAmbiguousNil(tags []TagReportData)
}

type TagReader struct {
	name string
	TProc
}

type ReaderGroup struct {
	mu       sync.RWMutex
	readers  map[string]*TagReader
	env      Environment
	behavior Behavior
}

func NewReaderGroup() *ReaderGroup {
	return &ReaderGroup{
		readers: map[string]*TagReader{},
		env:     Environment{},
		behavior: Behavior{
			ImpinjOptions: &ImpinjOptions{SuppressMonza: false},
			ScanType:      ScanNormal,
			Duration:      0,                      // infinite
			Power:         PowerTarget{Max: 3000}, // 30 dBm
			GPITrigger:    nil,
			Frequencies:   nil, // assume power is valid at all frequencies (for non-Hopping)
		},
	}
}

func (rg *ReaderGroup) Behavior() Behavior {
	return rg.behavior
}

func (rg *ReaderGroup) ProcessTagReport(name string, tags []TagReportData) bool {
	rg.mu.RLock()
	tr, ok := rg.readers[name]
	rg.mu.RUnlock()

	if !ok {
		return false
	}

	tr.FillAmbiguousNil(tags)
	return true
}

func (rg *ReaderGroup) RemoveReader(name string) {
	rg.mu.Lock()
	delete(rg.readers, name)
	rg.mu.Unlock()
}

func (rg *ReaderGroup) AddReader(ds DSClient, name string) error {
	r, err := ds.NewReader(name)
	if err != nil {
		return err
	}

	s, err := r.NewROSpec(rg.behavior, rg.env)
	if err != nil {
		return err
	}

	if err := r.replaceRO(ds, s); err != nil {
		return err
	}

	rg.mu.Lock()
	rg.readers[name] = r
	rg.mu.Unlock()

	return nil
}

func (r *TagReader) replaceRO(ds DSClient, spec *ROSpec) error {
	if err := ds.DeleteAllROSpecs(r.name); err != nil {
		return err
	}

	return ds.AddROSpec(r.name, spec)
}

// SetBehavior updates the behavior for each TagReader in the ReaderGroup.
func (rg *ReaderGroup) SetBehavior(ds DSClient, b Behavior) error {
	rg.mu.Lock()
	defer rg.mu.Unlock()

	specs := map[string]*ROSpec{}
	for name, r := range rg.readers {
		s, err := r.NewROSpec(b, rg.env)
		if err != nil {
			return errors.WithMessagef(err, "new behavior is invalid for %q", name)
		}
		specs[name] = s
	}

	// the behavior is valid for all members of the group
	rg.behavior = b

	errs := make(chan error, len(specs))
	wg := sync.WaitGroup{}
	wg.Add(len(specs))
	for d, s := range specs {
		go func(name string, s *ROSpec) {
			defer wg.Done()
			if err := rg.readers[name].replaceRO(ds, s); err != nil {
				errs <- errors.WithMessagef(err, "failed to replace ROSpec for %q", name)
			}
		}(d, s)
	}

	wg.Wait()
	close(errs)
	var errStrs []string
	for err := range errs {
		if err == nil {
			continue
		}

		errStrs = append(errStrs, err.Error())
	}

	errStr := strings.Join(errStrs, "; ")
	if errStr != "" {
		return errors.Errorf("failed to set some behaviors: %s", errStr)
	}

	return nil
}

type MultiErr []error

func (me MultiErr) Error() string {
	strs := make([]string, len(me))
	for i, s := range me {
		strs[i] = s.Error()
	}

	return strings.Join(strs, "; ")
}

func (rg *ReaderGroup) StartAll(ds DSClient) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	var errs []error
	for name := range rg.readers {
		if err := ds.EnableROSpec(name, 1); err != nil {
			errs = append(errs, err)
		}

		if rg.behavior.StartTrigger().Trigger == ROStartTriggerNone {
			if err := ds.StartROSpec(name, 1); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if errs != nil {
		return MultiErr(errs)
	}
	return nil
}

func (rg *ReaderGroup) StopAll(ds DSClient) error {
	rg.mu.RLock()
	defer rg.mu.RUnlock()

	var errs []error
	for name := range rg.readers {
		if err := ds.DisableROSpec(name, 1); err != nil {
			errs = append(errs, err)
		}

		if rg.behavior.StartTrigger().Trigger == ROStartTriggerNone {
			if err := ds.StopROSpec(name, 1); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if errs != nil {
		return MultiErr(errs)
	}
	return nil
}

func (ds DSClient) NewReader(device string) (*TagReader, error) {
	devCap, err := ds.GetCapabilities(device)
	if err != nil {
		return nil, err
	}

	if devCap.GeneralDeviceCapabilities == nil {
		return nil, errors.Errorf("missing general capabilities for %q", device)
	}

	var cr TProc
	switch devCap.GeneralDeviceCapabilities.DeviceManufacturer {
	case ImpinjPEN:
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

		cr = impDev
	default:
		basic, err := NewBasicDevice(devCap)
		if err != nil {
			return nil, err
		}

		if err := ds.SetConfig(device, basic.NewConfig()); err != nil {
			return nil, err
		}

		cr = basic
	}

	r := &TagReader{
		name:  device,
		TProc: cr,
	}

	return r, nil
}

func NewDSClient(host *url.URL, c *http.Client) DSClient {
	base := url.URL{
		Scheme: host.Scheme,
		Opaque: host.Opaque,
		User:   host.User,
		Host:   host.Host,
		Path:   "/api/v1/device/name/",
	}

	return DSClient{
		baseURL:    base.String(),
		httpClient: c,
	}
}

func (ds DSClient) GetCapabilities(device string) (*GetReaderCapabilitiesResponse, error) {
	r, err := ds.httpClient.Get(ds.baseURL + device + "/capabilities")
	if err != nil {
		return nil, errors.Wrap(err, "device info request failed")
	}

	if r.StatusCode != 200 {
		return nil, errors.Errorf("device info request failed with status %d", r.StatusCode)
	}

	defer r.Body.Close()
	const maxBody = 100 * 1024
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
		if reading.Name == "ReaderCapabilities" {
			caps = &GetReaderCapabilitiesResponse{}
			if err := json.Unmarshal([]byte(reading.Value), caps); err != nil {
				return nil, errors.Wrap(err, "device info request failed")
			}
			break
		}
	}

	if caps == nil {
		return nil, errors.New("failed to get ReaderCapabilities")
	}

	return caps, nil
}

func (ds DSClient) SetConfig(device string, conf *SetReaderConfig) error {
	confData, err := json.Marshal(conf)
	if err != nil {
		return errors.Wrap(err, "failed to marshal SetReaderConfig message")
	}

	edgexReq, err := json.Marshal(struct{ ReaderConfig string }{string(confData)})
	if err != nil {
		return errors.Wrap(err, "failed to marshal ReaderConfig edgex request")
	}

	req, err := http.NewRequest("PUT", ds.baseURL+device+"/config", bytes.NewReader(edgexReq))
	if err != nil {
		return errors.Wrap(err, "failed to create SetReaderConfig request")
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	r, err := ds.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to set ReaderConfig")
	}
	defer r.Body.Close()

	if !(200 <= r.StatusCode && r.StatusCode < 300) {
		return errors.Errorf("unexpected status code when setting config: %d", r.StatusCode)
	}

	return nil
}

func (ds DSClient) modifyROSpecState(state, device string, id uint32) error {
	edgexReq, err := json.Marshal(struct{ ROSpecID string }{strconv.FormatUint(uint64(id), 10)})
	if err != nil {
		return errors.Wrap(err, "failed to marshal ROSpec")
	}

	req, err := http.NewRequest("PUT", ds.baseURL+device+"/"+state+"ROSpec",
		bytes.NewReader(edgexReq))
	if err != nil {
		return errors.Wrap(err, "failed to create ROSpec request")
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	r, err := ds.httpClient.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to %s ROSpec", state)
	}
	defer r.Body.Close()

	if !(200 <= r.StatusCode && r.StatusCode < 300) {
		return errors.Errorf("unexpected status code: %d", r.StatusCode)
	}

	return nil
}

func (ds DSClient) EnableROSpec(device string, id uint32) error {
	return ds.modifyROSpecState("enable", device, id)
}

func (ds DSClient) DisableROSpec(device string, id uint32) error {
	return ds.modifyROSpecState("disable", device, id)
}

func (ds DSClient) StopROSpec(device string, id uint32) error {
	return ds.modifyROSpecState("stop", device, id)
}

func (ds DSClient) StartROSpec(device string, id uint32) error {
	return ds.modifyROSpecState("start", device, id)
}

func (ds DSClient) DeleteROSpec(device string, id uint32) error {
	return ds.modifyROSpecState("delete", device, id)
}

func (ds DSClient) DeleteAllROSpecs(device string) error {
	return ds.modifyROSpecState("delete", device, 0)
}

func (ds DSClient) AddROSpec(device string, spec *ROSpec) error {
	roData, err := json.Marshal(spec)
	if err != nil {
		return errors.Wrap(err, "failed to marshal ROSpec")
	}

	edgexReq, err := json.Marshal(struct{ ROSpec string }{string(roData)})
	if err != nil {
		return errors.Wrap(err, "failed to marshal ReaderConfig edgex request")
	}

	req, err := http.NewRequest("PUT", ds.baseURL+device+"/roSpec", bytes.NewReader(edgexReq))
	if err != nil {
		return errors.Wrap(err, "failed to create AddROSpec request")
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	r, err := ds.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to add ROSpec")
	}
	defer r.Body.Close()

	if !(200 <= r.StatusCode && r.StatusCode < 300) {
		return errors.Errorf("unexpected status code: %d", r.StatusCode)
	}

	return nil
}

func (d *ImpinjDevice) EnableCustomExt(name string, ds DSClient) error {
	req, err := http.NewRequest("PUT", ds.baseURL+name+"/enableImpinjExt",
		bytes.NewReader([]byte(`{"ImpinjCustomExtensionMessage":"AAAAAA=="}`)))
	if err != nil {
		return errors.Wrap(err, "failed to create request to enable Impinj custom extensions")
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to enable Impinj extensions")
	}
	defer resp.Body.Close()

	if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
