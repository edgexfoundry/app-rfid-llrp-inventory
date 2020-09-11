//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
)

const (
	serviceKey = "rfid-inventory"

	ResourceInventoryEvent     = "InventoryEvent"
	ResourceROAccessReport     = "ROAccessReport"
	ResourceReaderNotification = "ReaderEventNotification"

	// keys into the application settings map
	sKeyDevServiceName   = "DeviceServiceName"
	sKeyDeviceServiceURL = "DeviceServiceURL"
	sKeyMetaServiceURL   = "MetadataServiceURL"
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

var roSpecID0 = []byte(`{"ROSpecID":"0"}`)

func main() {
	edgexSdk := &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := edgexSdk.Initialize(); err != nil {
		panic(fmt.Sprintf("SDK initialization failed: %v\n", err))
	}

	lgr := logWrap{edgexSdk.LoggingClient}
	lgr.Info("Starting.")

	appSettings := edgexSdk.ApplicationSettings()
	lgr.exitIf(appSettings == nil, "Missing application settings.")

	metadataURI, err := url.Parse(strings.TrimSpace(appSettings[sKeyMetaServiceURL]))
	lgr.exitIfErr(err, "Invalid device service URL.")
	lgr.exitIf(metadataURI.Scheme == "" || metadataURI.Host == "",
		"Invalid metadata service URL.", lg{"endpoint", metadataURI.String()})

	devServURI, err := url.Parse(strings.TrimSpace(appSettings[sKeyDeviceServiceURL]))
	lgr.exitIfErr(err, "Invalid device service URL.")
	lgr.exitIf(devServURI.Scheme == "" || devServURI.Host == "",
		"Invalid device service URL.", lg{"endpoint", devServURI.String()})
	tagProc := inventory.NewTagProcessor(lgr)

	devService := llrp.NewDSClient(&url.URL{
		Scheme: devServURI.Scheme,
		Host:   devServURI.Host,
	}, http.DefaultClient)

	ep := newEventProc(lgr, tagProc, devService)

	dsName := strings.TrimSpace(appSettings[sKeyDevServiceName])
	lgr.exitIf(dsName == "", "Missing device service name.", lg{"key", sKeyDevServiceName})
	metadataURI.Path = "/api/v1/device/servicename/" + dsName
	deviceNames, err := internal.GetDevices(metadataURI.String(), http.DefaultClient)
	lgr.exitIfErr(err, "Failed to get existing device names.", lg{"path", metadataURI.String()})
	for _, name := range deviceNames {
		lgr.exitIfErr(ep.defaultGrp.AddReader(ep.devService, name),
			"Failed to setup device.", lg{"device", name})
	}

	// init routes
	for _, rte := range []struct {
		path, method string
		f            http.HandlerFunc // of course the EdgeX SDK doesn't take a http.Handler...
	}{
		{"/", http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "res/html/index.html")
		}},
		{"/api/v1/inventory/raw", http.MethodGet, internal.RawInventory(lgr, tagProc)},
		{"/api/v1/command/reading/start", http.MethodPut,
			func(w http.ResponseWriter, req *http.Request) {
				if err := ep.defaultGrp.StartAll(devService); err != nil {
					lgr.Error("Failed to StartAll.", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			},
		},
		{"/api/v1/command/reading/stop", http.MethodPut,
			func(w http.ResponseWriter, req *http.Request) {
				if err := ep.defaultGrp.StopAll(devService); err != nil {
					lgr.Error("Failed to StopAll.", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			},
		},
		{"/api/v1/behaviors/{name}", http.MethodGet,
			func(w http.ResponseWriter, req *http.Request) {
				rv := mux.Vars(req)
				bName := rv["name"]
				if bName != "default" {
					lgr.Error("Request to GET unknown behavior.", "name", bName)
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				data, err := json.Marshal(ep.defaultGrp.Behavior())
				if err != nil {
					lgr.Error("Failed to marshal behavior.", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if _, err := w.Write(data); err != nil {
					lgr.Error("Failed to write behavior data.", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
		},
		{"/api/v1/behaviors/{name}", http.MethodPut,
			func(w http.ResponseWriter, req *http.Request) {
				rv := mux.Vars(req)
				bName := rv["name"]
				if bName != "default" {
					lgr.Error("Attempt to PUT unknown behavior.", "name", bName)
					if _, err := w.Write([]byte("Invalid behavior name.")); err != nil {
						lgr.Error("Error writing failure response.", "error", err)
					}
					w.WriteHeader(http.StatusNotFound)
					return
				}

				data, err := ioutil.ReadAll(io.LimitReader(req.Body, 100*1024))
				if err != nil {
					lgr.Error("Failed to read behavior body.", "error", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				var b llrp.Behavior
				if err := json.Unmarshal(data, &b); err != nil {
					lgr.Error("Failed to unmarshal behavior body.", "error", err,
						"body", string(data))
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				if err := ep.defaultGrp.SetBehavior(devService, b); err != nil {
					lgr.Error("Failed to set new behavior.", "error", err)
					if _, err := w.Write([]byte(err.Error())); err != nil {
						lgr.Error("Error writing failure response.", "error", err)
					}
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				lgr.Info("Updated behavior.", "name", bName)
			},
		},
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
		if len(params) == 2 {
			if s, ok := params[1].(string); ok && s == "" {
				// Turns out, sometimes the "pipeline" gives a second parameter:
				// an empty string which sometimes has type info about the first param.
				// Too bad we aren't using a compiled language with static typing.
			} else {
				return false, errors.Errorf("expected a single parameter, but got a second: %T %[1]+v", params[1])
			}
		} else {
			return false, errors.Errorf("expected a single parameter, but got %d", len(params))
		}
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
			ep.lgr.Debug("Unknown reading.", "reading", reading.Name)
			continue

		case ResourceReaderNotification:
			r.Reset()
			r.WriteString(reading.Value)
			notification := &llrp.ReaderEventNotification{}
			if err := decoder.Decode(notification); err != nil {
				ep.lgr.Error("Failed to decode reader event notification", "error", err.Error())
				continue
			}

			if err := ep.handleReaderEvent(event.Device, notification); err != nil {
				ep.lgr.Error("Failed to handle ReaderEventNotification.",
					"error", err.Error(), "device", event.Device)
			}

		case ResourceROAccessReport:
			ep.lgr.Info("Processing RO.")
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
	lgr     logWrap
	tagProc *inventory.TagProcessor

	devMu      sync.RWMutex
	devService llrp.DSClient
	defaultGrp *llrp.ReaderGroup
}

func newEventProc(lgr logWrap, tagProc *inventory.TagProcessor, devService llrp.DSClient) *eventProc {
	return &eventProc{
		lgr:        lgr,
		tagProc:    tagProc,
		devService: devService,
		defaultGrp: llrp.NewReaderGroup(),
	}
}

func (ep *eventProc) setDefaultBehavior(b llrp.Behavior) error {
	ep.devMu.Lock()
	err := ep.defaultGrp.SetBehavior(ep.devService, b)
	ep.devMu.Unlock()
	return err
}

func (ep *eventProc) handleROAccessReport(edgeXCtx *appcontext.Context, device string, report *llrp.ROAccessReport) error {
	if report.TagReportData == nil {
		ep.lgr.Warn("No tag report data in report.")
		return nil
	}

	// This fills in ambiguous nil values.
	if !ep.defaultGrp.ProcessTagReport(device, report.TagReportData) {
		// This can only happen if the device didn't exist when we started,
		// and we never got a Connection message for it.
		return errors.Errorf("unknown device: %q", device)
	}

	for _, td := range report.TagReportData {
		if len(td.EPC96.EPC) != (96 / 8) {
			ep.lgr.Debug("Skipping non-EPC96.")
			continue
		}

		epc := hex.EncodeToString(td.EPC96.EPC)

		var antID, rssi int
		if td.AntennaID != nil {
			antID = int(*td.AntennaID)
		} else {
			ep.lgr.Warn("Missing AntennaID.", "EPC96", epc)
		}

		if td.PeakRSSI != nil {
			rssi = int(*td.PeakRSSI)
		} else {
			ep.lgr.Warn("Missing PeakRSSI.", "EPC96", epc)
		}

		var timestamp int64
		if td.LastSeenUTC != nil {
			timestamp = int64(*td.LastSeenUTC)
		} else {
			ep.lgr.Warn("Missing Timestamp.", "EPC96", epc)
		}

		gen2Read := inventory.Gen2Read{
			EPC:       epc,
			DeviceID:  device,
			AntennaID: antID,
			RSSI:      rssi,
			Timestamp: timestamp,
		}

		e := ep.tagProc.ProcessReadData(&gen2Read)
		if e == nil {
			continue
		}

		ep.lgr.Debug("Processing event.", "eventType", e.OfType(), "event", fmt.Sprintf("%+v", e))

		payload, err := json.Marshal(e)
		if err != nil {
			ep.lgr.Error("Failed to marshal output", "EPC96", epc, "error", err.Error())
			continue
		}

		eventName := ResourceInventoryEvent + e.OfType()
		if _, err := edgeXCtx.PushToCoreData(device, eventName, string(payload)); err != nil {
			ep.lgr.Error("Failed to push to Core Data", "EPC96", epc, "error", err.Error())
		}
	}

	return nil
}

func (ep *eventProc) handleReaderEvent(device string, notification *llrp.ReaderEventNotification) error {
	const connSuccess = llrp.ConnectionAttemptEvent(llrp.ConnSuccess)

	data := notification.ReaderEventNotificationData
	switch {
	case data.ConnectionAttemptEvent != nil && *data.ConnectionAttemptEvent == connSuccess:
		return ep.defaultGrp.AddReader(ep.devService, device)

	case data.ConnectionCloseEvent != nil:
		ep.defaultGrp.RemoveReader(device)
	}

	return nil
}
