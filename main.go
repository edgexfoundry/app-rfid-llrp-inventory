//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/edgexfoundry/go-mod-bootstrap/bootstrap/flags"
	"github.com/edgexfoundry/go-mod-configuration/configuration"
	"github.com/edgexfoundry/go-mod-configuration/pkg/types"
	"github.com/edgexfoundry/go-mod-core-contracts/clients"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/inventory"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	serviceKey      = "rfid-inventory"
	eventDeviceName = "rfid-inventory"

	BaseConsulPath = "edgex/appservices/1.0/"

	ResourceROAccessReport         = "ROAccessReport"
	ResourceReaderNotification     = "ReaderEventNotification"
	ResourceInventoryEventArrived  = "InventoryEventArrived"
	ResourceInventoryEventMoved    = "InventoryEventMoved"
	ResourceInventoryEventDeparted = "InventoryEventDeparted"

	// keys into the application settings map
	sKeyDevServiceName   = "DeviceServiceName"
	sKeyDeviceServiceURL = "DeviceServiceURL"
	sKeyMetaServiceURL   = "MetadataServiceURL"

	maxBodyBytes = 100 * 1024
)

type inventoryApp struct {
	edgexSdk   *appsdk.AppFunctionsSDK
	sdkCtx     atomic.Value
	lgr        logWrap
	devMu      sync.RWMutex
	devService llrp.DSClient
	defaultGrp *llrp.ReaderGroup

	snapshotReqs chan snapshotDest
	reports      chan reportData
}

type reportData struct {
	report *llrp.ROAccessReport
	info   inventory.ReportInfo
}

type snapshotDest struct {
	w      io.Writer
	result chan error
}

type consulConfig struct {
	Aliases map[string]string
}

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

func main() {
	edgexSdk := &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := edgexSdk.Initialize(); err != nil {
		panic(fmt.Sprintf("SDK initialization failed: %v\n", err))
	}

	lgr := logWrap{edgexSdk.LoggingClient}
	lgr.Info("Starting.")

	appSettings := edgexSdk.ApplicationSettings()
	lgr.exitIf(appSettings == nil, "Missing application settings.")
	cc, err := getConfigClient()
	lgr.exitIfErr(err, "Failed to create config client.")

	metadataURI, err := url.Parse(strings.TrimSpace(appSettings[sKeyMetaServiceURL]))
	lgr.exitIfErr(err, "Invalid device service URL.")
	lgr.exitIf(metadataURI.Scheme == "" || metadataURI.Host == "",
		"Invalid metadata service URL.", lg{"endpoint", metadataURI.String()})

	devServURI, err := url.Parse(strings.TrimSpace(appSettings[sKeyDeviceServiceURL]))
	lgr.exitIfErr(err, "Invalid device service URL.")
	lgr.exitIf(devServURI.Scheme == "" || devServURI.Host == "",
		"Invalid device service URL.", lg{"endpoint", devServURI.String()})

	defaultGrp := llrp.NewReaderGroup()
	devService := llrp.NewDSClient(&url.URL{
		Scheme: devServURI.Scheme,
		Host:   devServURI.Host,
	}, http.DefaultClient)

	dsName := strings.TrimSpace(appSettings[sKeyDevServiceName])
	lgr.exitIf(dsName == "", "Missing device service name.", lg{"key", sKeyDevServiceName})
	metadataURI.Path = "/api/v1/device/servicename/" + dsName
	deviceNames, err := llrp.GetDevices(metadataURI.String(), http.DefaultClient)
	lgr.exitIfErr(err, "Failed to get existing device names.", lg{"path", metadataURI.String()})
	for _, name := range deviceNames {
		lgr.exitIfErr(defaultGrp.AddReader(devService, name),
			"Failed to setup device.", lg{"device", name})
	}

	app := inventoryApp{
		lgr:          lgr,
		defaultGrp:   defaultGrp,
		devService:   devService,
		snapshotReqs: make(chan snapshotDest),
		reports:      make(chan reportData),
	}

	// routes
	for _, rte := range []struct {
		path, method string
		f            http.HandlerFunc // of course the EdgeX SDK doesn't take a http.Handler...
	}{
		{"/", http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "res/html/index.html")
		}},
		{"/api/v1/readers", http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if err := app.defaultGrp.ListReaders(w); err != nil {
				app.lgr.Error("Failed to write readers list.", "error", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
		}},
		{"/api/v1/inventory/snapshot", http.MethodGet,
			func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if err := app.writeInventorySnapshot(w); err != nil {
					app.lgr.Error("Failed to write inventory snapshot.", "error", err.Error())
					w.WriteHeader(http.StatusInternalServerError)
				}
			},
		},
		{"/api/v1/command/reading/start", http.MethodPost,
			func(w http.ResponseWriter, req *http.Request) {
				if err := app.defaultGrp.StartAll(devService); err != nil {
					lgr.Error("Failed to StartAll.", "error", err.Error())
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			},
		},
		{"/api/v1/command/reading/stop", http.MethodPost,
			func(w http.ResponseWriter, req *http.Request) {
				if err := app.defaultGrp.StopAll(devService); err != nil {
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

				data, err := json.Marshal(app.defaultGrp.Behavior())
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

				data, err := ioutil.ReadAll(io.LimitReader(req.Body, maxBodyBytes))
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

				if err := app.defaultGrp.SetBehavior(devService, b); err != nil {
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

	var wg sync.WaitGroup
	wg.Add(1)
	done := make(chan struct{})
	go func() {
		defer wg.Done()
		app.taskLoop(done, *cc, lgr)
		lgr.Info("Done processing.")
	}()

	// Subscribe to events.
	lgr.exitIfErr(edgexSdk.SetFunctionsPipeline(app.processEdgeXEvent), "Failed to build pipeline.")
	lgr.exitIfErr(edgexSdk.MakeItRun(), "Failed to run pipeline.")

	// let task loop complete
	close(done)
	wg.Wait()
}

// getConfigClient returns a configuration client based on the command line args,
// or a default one if those lack a config provider URL.
// Ideally, a future version of the EdgeX SDKs will give us something like this
// without parsing the args again, but for now, this will do.
func getConfigClient() (*configuration.Client, error) {
	sdkFlags := flags.New()
	sdkFlags.Parse(os.Args[1:])
	cpUrl, err := url.Parse(sdkFlags.ConfigProviderUrl())
	if err != nil {
		return nil, err
	}

	cpPort := 8500
	port := cpUrl.Port()
	if port != "" {
		cpPort, err = strconv.Atoi(port)
		if err != nil {
			return nil, errors.Wrap(err, "bad config port")
		}
	}

	configClient, err := configuration.NewConfigurationClient(types.ServiceConfig{
		Host:     cpUrl.Hostname(),
		Port:     cpPort,
		BasePath: BaseConsulPath,
		Type:     cpUrl.Scheme,
	})

	return &configClient, errors.Wrap(err, "failed to get config client")
}

// processEdgeXEvent is used as the sole member of our pipeline.
// It's essentially our entrypoint for EdgeX event processing.
//
// Using the pipeline SDK is the least-effort method
// of accomplishing the grunt work of
// subscribing to EdgeX's event stream and
// accessing the resources that its agnosticism necessitates
// may come from any of several sources.
//
// But since it's a lot easier, safer, and more performant
// to write, call, compose, and test typical Go functions,
// we only use the SDK to call a single function (this one),
// which must verify the parameter types and arity,
// then verify the safety we lost by piping this through EdgeX by
// string matching the Event.Reading[].Name and JSON-unmarshal the Value string.
//
// Once we've reestablished these basic requirements,
// this dispatches the content to the appropriate type-safe functions.
func (app *inventoryApp) processEdgeXEvent(edgeXCtx *appcontext.Context, params ...interface{}) (bool, interface{}) {
	app.sdkCtx.Store(edgeXCtx)

	if len(params) != 1 {
		if len(params) == 2 {
			if s, ok := params[1].(string); ok && s == "" {
				// Turns out, sometimes the "pipeline" gives a second parameter:
				// an empty string which sometimes has type info about the first param.
			} else {
				err := errors.Errorf("expected a single parameter, but got a second: %T %+[1]v", params[1])
				app.lgr.Error("Processing error.", "error", err.Error())
				return false, err
			}
		} else {
			err := errors.Errorf("expected a single parameter, but got %d", len(params))
			app.lgr.Error("Processing error.", "error", err.Error())
			return false, err
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
		reading := &event.Readings[i] // Readings is 169 bytes. This avoid the copy.
		switch reading.Name {
		default:
			app.lgr.Debug("Unknown reading.", "reading", reading.Name)
			continue

		case ResourceReaderNotification:
			r.Reset()
			r.WriteString(reading.Value)
			notification := &llrp.ReaderEventNotification{}
			if err := decoder.Decode(notification); err != nil {
				app.lgr.Error("Failed to decode reader event notification", "error", err.Error())
				continue
			}

			if err := app.handleReaderEvent(event.Device, notification); err != nil {
				app.lgr.Error("Failed to handle ReaderEventNotification.",
					"error", err.Error(), "device", event.Device)
			}

		case ResourceROAccessReport:
			r.Reset()
			r.WriteString(reading.Value)

			report := &llrp.ROAccessReport{}
			if err := decoder.Decode(report); err != nil {
				app.lgr.Error("Failed to decode tag report",
					"error", err.Error(), "device", event.Device)
				continue
			}

			if report.TagReportData == nil {
				app.lgr.Warn("No tag report data in report.", "device", event.Device)
			} else {
				app.reports <- reportData{report, inventory.NewReportInfo(reading)}
				app.lgr.Info("New ROAccessReport.",
					"device", event.Device, "tags", len(report.TagReportData))
			}
		}
	}

	return false, nil
}

// handleReaderEvent handles an llrp.ReaderEventNotification from the Device Service.
//
// If a device reports a new connection event,
// this adds the reader to the list of managed readers.
// If a device reports a close event, it removes that reader.
func (app *inventoryApp) handleReaderEvent(device string, notification *llrp.ReaderEventNotification) error {
	const connSuccess = llrp.ConnectionAttemptEvent(llrp.ConnSuccess)

	data := notification.ReaderEventNotificationData
	switch {
	case data.ConnectionAttemptEvent != nil && *data.ConnectionAttemptEvent == connSuccess:
		return app.defaultGrp.AddReader(app.devService, device)

	case data.ConnectionCloseEvent != nil:
		app.defaultGrp.RemoveReader(device)
	}

	return nil
}

// writeInventorySnapshot writes the current inventory snapshot to w.
func (app *inventoryApp) writeInventorySnapshot(w io.Writer) error {
	// We send w and a writeErr channel into the inventory execution context
	// and then wait to read a value from the writeErr channel.
	// That context closes writeErr to signal the snapshot is written to w
	// or an error prevented such, and we can send the result back to our caller.
	writeErr := make(chan error, 1)
	app.snapshotReqs <- snapshotDest{w, writeErr}
	return <-writeErr
}

// taskLoop is our main event loop for async processes
// that can't be modeled within the SDK's pipeline event loop.
//
// Namely, it launches scheduled tasks and configuration changes.
// Since nearly every round through this loop must read or write the inventory,
// this taskLoop ensures the modifications are done safely
// without requiring a ton of lock contention on the inventory itself.
func (app *inventoryApp) taskLoop(done chan struct{}, cc configuration.Client, lc logger.LoggingClient) {
	aggregateDepartedTicker := time.NewTicker(time.Duration(inventory.DepartedCheckIntervalSeconds) * time.Second)
	ageoutTicker := time.NewTicker(1 * time.Hour)
	confErrs := make(chan error)
	confUpdates := make(chan interface{})

	defer func() {
		close(confErrs)
		close(confUpdates)
	}()

	// load tag data
	var snapshot []inventory.StaticTag
	snapshotData, err := ioutil.ReadFile(inventory.TagCacheFile)
	if err != nil {
		lc.Warn("Failed to load inventory snapshot.", "error", err.Error())
	} else {
		if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
			lc.Warn("Failed to unmarshal inventory snapshot.", "error", err.Error())
		}
	}
	processor := inventory.NewTagProcessor(lc, snapshot)

	config := &consulConfig{
		Aliases: make(map[string]string),
	}
	cc.WatchForChanges(confUpdates, confErrs, config, "/"+serviceKey)

	lc.Info("Starting event loop.")

	for {
		var updatedSnapshot []inventory.StaticTag
		select {
		case <-done:
			return

		case rd := <-app.reports:
			lc.Info("New report")
			if !app.defaultGrp.ProcessTagReport(rd.info.DeviceName, rd.report.TagReportData) {
				// This can only happen if the device didn't exist when we started,
				// and we never got a Connection message for it.
				lc.Error("Tag Report for unknown device", "device", rd.info.DeviceName)
			}
			// todo: the deep scan status needs to be ascertained on a per-tag basis
			//       depending on whether or not the tag's existing device location is performing
			//       a deep scan. However for now we only support a single reader group so we can
			//       apply the logic as a single check.
			rd.info.IsDeepScan = app.defaultGrp.IsDeepScan()

			var events []inventory.Event
			events, updatedSnapshot = processor.ProcessReport(rd.report, rd.info)
			if err := app.pushEventsToCoreData(events); err != nil {
				lc.Error("Failed to push events to CoreData", "error", err.Error())
			}

		case t := <-aggregateDepartedTicker.C:
			_, ok := app.sdkCtx.Load().(*appcontext.Context)
			if !ok {
				lc.Warn("Delaying AggregateDeparted processor: missing app-functions-sdk context")
				break
			}
			lc.Debug("Running AggregateDeparted.", "time", fmt.Sprintf("%v", t))
			var events []inventory.Event
			events, updatedSnapshot = processor.AggregateDeparted()
			if err := app.pushEventsToCoreData(events); err != nil {
				lc.Error("Failed to push events to CoreData", "error", err.Error())
			}

		case t := <-ageoutTicker.C:
			lc.Debug("Running AgeOut.", "time", fmt.Sprintf("%v", t))
			_, updatedSnapshot = processor.AgeOut()

		case rawConfig := <-confUpdates:
			if newConfig, ok := rawConfig.(*consulConfig); ok {
				lc.Info("Configuration updated.")
				lc.Debug("New configuration", "raw", fmt.Sprintf("%+v", newConfig))
				config = newConfig
				processor.SetAliases(newConfig.Aliases)
			} else {
				lc.Warn("Unable to decode configuration.", "raw", fmt.Sprintf("%+v", rawConfig))
			}

		case req := <-app.snapshotReqs:
			_, err := req.w.Write(snapshotData)
			req.result <- err

		case err := <-confErrs:
			lc.Error("Configuration error.", "error", err.Error())
		}

		if updatedSnapshot == nil {
			continue
		}

		snapshot = updatedSnapshot

		lc.Info("Updating inventory snapshot.")
		data, err := json.Marshal(snapshot)
		if err != nil {
			lc.Warn("Failed to marshal inventory snapshot.", "error", err.Error())
			continue
		}

		snapshotData = data
		if err := ioutil.WriteFile(inventory.TagCacheFile, data, 0644); err != nil {
			lc.Warn("Failed to persist inventory snapshot.", "error", err.Error())
			continue
		}
		lc.Debug("Persisted inventory snapshot.", "tags", len(snapshot))
	}
}

// setDefaultBehavior sets the behavior associated with the default device group.
func (app *inventoryApp) setDefaultBehavior(b llrp.Behavior) error {
	app.devMu.Lock()
	err := app.defaultGrp.SetBehavior(app.devService, b)
	app.devMu.Unlock()
	return err
}

// pushEventsToCoreData will send one or more Inventory Events as a single EdgeX Event with
// an EdgeX Reading for each Inventory Event
func (app *inventoryApp) pushEventsToCoreData(events []inventory.Event) error {
	sdkCtx, ok := app.sdkCtx.Load().(*appcontext.Context)
	if !ok {
		return errors.New("unable to push events to core data: missing app-functions-sdk context")
	}

	now := time.Now().UnixNano()
	readings := make([]models.Reading, 0, len(events))

	var errs []error
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "error marshalling event"))
			continue
		}

		var resource string
		switch event.OfType() {
		case inventory.ArrivedType:
			resource = ResourceInventoryEventArrived
		case inventory.MovedType:
			resource = ResourceInventoryEventMoved
		case inventory.DepartedType:
			resource = ResourceInventoryEventDeparted
		default:
			errs = append(errs, errors.New("unknown event type!"))
			continue
		}

		app.edgexSdk.LoggingClient.Info("processing event",
			"type", event.OfType(), "payload", string(payload))

		readings = append(readings, models.Reading{
			Value:  string(payload),
			Origin: now,
			Device: eventDeviceName,
			Name:   resource,
		})
	}

	edgeXEvent := &models.Event{
		Device:   eventDeviceName,
		Origin:   now,
		Readings: readings,
	}

	correlation := uuid.New().String()
	ctx := context.WithValue(context.Background(), clients.CorrelationHeader, correlation)
	// todo: Once this issue (https://github.com/edgexfoundry/app-functions-sdk-go/issues/446) is
	//       resolved, we can use the appsdk.AppFunctionsSDK EventClient directly without the need
	//       for the appcontext.Context.
	if _, err := sdkCtx.EventClient.Add(ctx, edgeXEvent); err != nil {
		errs = append(errs, errors.Wrap(err, "unable to push inventory event(s) to core-data"))
	}

	if errs != nil {
		return llrp.MultiErr(errs)
	}
	return nil
}
