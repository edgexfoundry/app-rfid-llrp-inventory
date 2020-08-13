//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/commands/routes"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	serviceKey = "rfid-inventory"

	ResourceInventoryEvent     = "InventoryEvent"
	ResourceROAccessReport     = "ROAccessReport"
	ResourceReaderNotification = "ReaderEventNotification"

	// CoreCommandPUTDevice app settings
	CoreCommandPUTDevice = "CoreCommandPUTDevice"
	// CoreCommandGETDevices app settings
	CoreCommandGETDevices = "CoreCommandGETDevices"

	deviceServiceURL = "DeviceService"
)

type logWrap struct {
	logger.LoggingClient
}

type lg struct {
	key string
	val interface{}
}

func (lgr logWrap) errIf(cond bool, msg string, params ...lg) bool {
	if !cond {
		return false
	}

	if len(params) > 0 {
		parts := make([]interface{}, len(params)*2)
		for i := range params {
			parts[i*2] = params[i].key
			parts[i*2+1] = params[i].val
		}
		lgr.Error(msg, parts...)
	} else {
		lgr.Error(msg)
	}

	return true
}

func (lgr logWrap) exitIf(cond bool, msg string, params ...lg) {
	if lgr.errIf(cond, msg, params...) {
		os.Exit(1)
	}
}

func (lgr logWrap) exitIfErr(err error, msg string, params ...lg) {
	lgr.exitIf(err != nil, msg, append(params, lg{"error", err})...)
}

var roSpecID0 = []byte(`{"ROSpecID":0}`)

func main() {
	edgexSdk := &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := edgexSdk.Initialize(); err != nil {
		panic(fmt.Sprintf("SDK initialization failed: %v\n", err))
	}

	lgr := logWrap{edgexSdk.LoggingClient}
	lgr.Info("Starting.")
	tagProc := inventory.NewTagProcessor(lgr)
	ep := &eventProc{
		lgr:     lgr,
		tagProc: tagProc,
	}

	appSettings := edgexSdk.ApplicationSettings()
	lgr.exitIf(appSettings == nil, "Missing application settings.")

	devServURI, err := url.Parse(strings.TrimSpace(appSettings[deviceServiceURL]))
	lgr.exitIfErr(err, "Invalid device service URL.")
	lgr.exitIf(devServURI.Scheme == "" || devServURI.Host == "",
		"Invalid device service URL.", lg{"endpoint", devServURI.String()})

	devServURI.Path = "/api/v1/device/all/enableROSpec"
	startAllProxy := routes.NewPutProxy(lgr, devServURI.String(), roSpecID0)
	devServURI.Path = "/api/v1/device/all/disableROSpec"
	stopAllProxy := routes.NewPutProxy(lgr, devServURI.String(), roSpecID0)

	// init routes
	for _, rte := range []struct {
		path, method string
		f            http.HandlerFunc // of course the EdgeX SDK doesn't take a http.Handler...
	}{
		{"/", http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "res/html/index.html")
		}},
		{"/api/v1/inventory/raw", http.MethodGet, routes.RawInventory(lgr, tagProc)},
		{"/api/v1/command/reading/start", http.MethodPut, startAllProxy.HandleRequest},
		{"/api/v1/command/reading/stop", http.MethodPut, stopAllProxy.HandleRequest},
		{"/api/v1/command/behaviors/{behaviorCommand}", http.MethodPut, routes.SetBehaviors()},
	} {
		lgr.exitIfErr(edgexSdk.AddRoute(rte.path, rte.f, rte.method),
			"Failed to add route.", lg{"path", rte.path}, lg{"method", rte.method})
	}

	// Subscribe to events.
	lgr.exitIfErr(edgexSdk.SetFunctionsPipeline(ep.processEdgeXEvent), "Failed to build pipeline.")
	lgr.exitIfErr(edgexSdk.MakeItRun(), "Failed to run pipeline.")
}

func (ep *eventProc) processEdgeXEvent(edgeXCtx *appcontext.Context, params ...interface{}) (bool, interface{}) {
	if len(params) != 1 {
		return false, errors.Errorf("expected a single parameter, but got %d", len(params))
	}

	event, ok := params[0].(models.Event)
	if !ok {
		// You know what's cool in compiled languages? Type safety.
		return false, errors.Errorf("expected an EdgeX Event, but got %T", event)
	}

	if len(event.Readings) < 1 {
		// Is this really an error? EdgeX's Filter functions say yes.
		return false, errors.New("event contains no Readings")
	}

	r := bytes.Buffer{}
	decoder := json.NewDecoder(&r)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()

	for i := range event.Readings {
		reading := &event.Readings[i] // Readings is 169 bytes. Let's avoid the copy.
		switch reading.Name {
		default:
			continue

		case ResourceReaderNotification:
			r.Reset()
			r.WriteString(reading.Value)
			notification := &llrp.ReaderEventNotification{}
			if err := decoder.Decode(notification); err != nil {
				ep.lgr.Error("Failed to decode reader event notification", "error", err.Error())
				continue
			}

			ep.handleReaderEvent(event.Device, notification)

		case ResourceROAccessReport:
			r.Reset()
			r.WriteString(reading.Value)

			report := &llrp.ROAccessReport{}
			if err := decoder.Decode(report); err != nil {
				ep.lgr.Error("Failed to decode tag report",
					"error", err.Error(), "device", event.Device)
				continue
			}

			if err := ep.handleROAccessReport(edgeXCtx, event.Device, report); err != nil {
				ep.lgr.Error("Failed to process ROAccessReport.",
					"error", err.Error(), "device", event.Device)
			}
		}
	}

	return false, nil
}

type eventProc struct {
	lgr         logWrap
	tagProc     *inventory.TagProcessor
	deviceMu    sync.RWMutex
	reportSpecs map[string]*device
}

type device struct {
	connected   time.Time
	generalCap  llrp.GeneralDeviceCapabilities
	powerLevels []llrp.TransmitPowerLevelTableEntry
	channels    []llrp.FrequencyInformation
	report      llrp.TagReportContentSelector
	last        llrp.TagReportData
}

func (ep *eventProc) getReportSpec(deviceName string) (s *device) {
	ep.deviceMu.RLock()
	s = ep.reportSpecs[deviceName]
	ep.deviceMu.RUnlock()
	return
}

func (ep *eventProc) setReportSpec(deviceName string, s *device) {
	ep.deviceMu.Lock()
	ep.reportSpecs[deviceName] = s
	ep.deviceMu.Unlock()
	return
}

var (
	ErrUnknownReportSpec = fmt.Errorf("unknown report spec")
)

func (ep *eventProc) handleROAccessReport(edgeXCtx *appcontext.Context, device string, report *llrp.ROAccessReport) error {
	if report.TagReportData == nil {
		return nil
	}

	// LLRP has a data compression "feature" that allows Readers to omit some parameters
	// if the value hasn't changed "since the last time it was sent".
	// This has several unfortunate consequences when it comes to proper processing.
	// For now, we'll assume that we use a single, consistent report spec (per Reader)
	// and that we receive & process reports in the order the Reader sent them.
	// Although they don't document it, Impinj told us they always send all parameters,
	// so we won't even bother with this for Impinj Readers.
	// For others, it only matters for the parameters we care about/enable.
	s := ep.getReportSpec(device)
	if s == nil {
		return ErrUnknownReportSpec
	}

	for i := range report.TagReportData {
		tagData := &report.TagReportData[i]
		s.last

	}

	gen2Read := inventory.Gen2Read{
		EPC:       "",
		TID:       "",
		User:      "",
		Reserved:  "",
		DeviceID:  "",
		AntennaID: 0,
		Timestamp: 0,
		RSSI:      0,
	}

	e := ep.tagProc.ProcessReadData(&gen2Read)
	if e == nil {
		return
	}

	ep.lgr.Debug("Processing event.", "eventType", e.OfType(), "event", fmt.Sprintf("%+v", e))

	payload, err := json.Marshal(e)
	if err != nil {
		ep.lgr.Error("Failed to marshal output event.",
			"eventType", e.OfType(), "error", err.Error())
		return
	}

	eventName := ResourceInventoryEvent + e.OfType()
	if _, err := edgeXCtx.PushToCoreData(device, eventName, string(payload)); err != nil {
		ep.lgr.Error("Failed to push inventory event to core-data.", "error", err.Error())
	}
}

func (ep *eventProc) handleReaderEvent(device string, notification *llrp.ReaderEventNotification) {
	if notification.ReaderEventNotificationData.ConnectionAttemptEvent != nil {
		cae := llrp.ConnectionAttemptEventType(*notification.ReaderEventNotificationData.ConnectionAttemptEvent)
		if cae == llrp.ConnSuccess {
			ep.configureReader(device)
		}
	} else if notification.ReaderEventNotificationData.ConnectionCloseEvent != nil {

	}
}

func (ep *eventProc) configureReader(device string) {

}
