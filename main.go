//
// Copyright (c) 2020 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package main

import (
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/edgexfoundry/app-functions-sdk-go/pkg/transforms"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/inventory"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/routes"
	"golang.org/x/net/context"
	"net/http"
	"os"
	"strings"
	"sync"
)

const (
	serviceKey    = "rfid-inventory"
	readChBuffSz  = 1000
	eventChBuffSz = 10

	ResourceROAccessReport = "ROAccessReport"
	ResourceInventoryEvent = "inventory_event"

	LLRPDeviceService = "LLRPDeviceService"
)

type inventoryApp struct {
	edgexSdk        *appsdk.AppFunctionsSDK
	edgexSdkContext *appcontext.Context

	processor *inventory.TagProcessor
	readCh    chan *inventory.AccessReport
	eventCh   chan *jsonrpc.InventoryEvent

	done chan struct{}
}

func main() {

	app := inventoryApp{}
	// initialize Edgex App functions SDK
	app.edgexSdk = &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := app.edgexSdk.Initialize(); err != nil {
		if app.edgexSdk.LoggingClient == nil {
			fmt.Printf("SDK initialization failed: %v\n", err)
		} else {
			app.edgexSdk.LoggingClient.Error(fmt.Sprintf("SDK initialization failed: %v\n", err))
		}
		os.Exit(-1)
	}
	app.done = make(chan struct{})
	app.readCh = make(chan *inventory.AccessReport, readChBuffSz)
	app.eventCh = make(chan *jsonrpc.InventoryEvent, eventChBuffSz)
	app.processor = inventory.NewTagProcessor(app.edgexSdk.LoggingClient)
	app.edgexSdk.LoggingClient.Info(fmt.Sprintf("Running"))

	// Retrieve the application settings from configuration.toml
	appSettings := app.edgexSdk.ApplicationSettings()
	if appSettings == nil {
		app.edgexSdk.LoggingClient.Error("No application settings found")
		os.Exit(-1)
	}

	// Create SettingsHandler struct with logger & appsettings to be passed to http response context object
	settingsHandlerVar := routes.SettingsHandler{Logger: app.edgexSdk.LoggingClient, AppSettings: appSettings}

	err := app.edgexSdk.AddRoute("/", passSettings(settingsHandlerVar, routes.Index), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/ping", passSettings(settingsHandlerVar, routes.Ping), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/inventory/raw", passSettings(settingsHandlerVar, routes.RawInventory), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/command/readers", passSettings(settingsHandlerVar, routes.GetDevices), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/command/readings/{readCommand}", passSettings(settingsHandlerVar, routes.IssueReadOrStop), http.MethodPost)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/command/behaviors/{behaviorCommand}", passSettings(settingsHandlerVar, routes.IssueBehavior), http.MethodPut)
	addRouteErrorHandler(app.edgexSdk, err)

	// the collection of functions to execute every time an event is triggered.
	err = app.edgexSdk.SetFunctionsPipeline(
		app.contextGrabber,
		transforms.NewFilter([]string{ResourceROAccessReport}).FilterByValueDescriptor,
		app.processEvents,
	)
	if err != nil {
		app.edgexSdk.LoggingClient.Error("Error in the pipeline: ", err.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go app.processReadChannel(&wg)
	wg.Add(1)
	go app.processEventChannel(&wg)

	// tell SDK to "start" and begin listening for events to trigger the pipeline.
	err = app.edgexSdk.MakeItRun()
	if err != nil {
		app.edgexSdk.LoggingClient.Error("MakeItRun returned error: ", err.Error())
		os.Exit(-1)
	}

	app.edgexSdk.LoggingClient.Info("waiting for channels to finish")
	close(app.done)
	wg.Wait()

	// Do any required cleanup here
	os.Exit(0)
}

// contextGrabber does what it sounds like, it grabs the app-functions-sdk's appcontext.Context. This is needed
// because the context is not available outside of a pipeline without using reflection and unsafe pointers
func (app *inventoryApp) contextGrabber(edgexContext *appcontext.Context, params ...interface{}) (bool, interface{}) {
	if app.edgexSdkContext == nil {
		app.edgexSdkContext = edgexContext
		app.edgexSdk.LoggingClient.Debug("grabbed app-functions-sdk context")
	}

	if len(params) < 1 {
		return false, errors.New("no event received")
	}

	existingEvent, ok := params[0].(models.Event)
	if !ok {
		return false, errors.New("type received is not an Event")
	}

	return true, existingEvent
}

func (app *inventoryApp) processEvents(_ *appcontext.Context, params ...interface{}) (bool, interface{}) {

	if len(params) < 1 {
		return false, errors.New("no event received")
	}
	event, ok := params[0].(models.Event)
	if !ok {
		return false, errors.New("type received is not an Event")
	}
	if len(event.Readings) < 1 {
		return false, errors.New("event contains no Readings")
	}

	for _, reading := range event.Readings {
		switch reading.Name {

		case ResourceROAccessReport:
			report := llrp.ROAccessReport{}
			decoder := json.NewDecoder(strings.NewReader(reading.Value))
			decoder.UseNumber()

			if err := decoder.Decode(&report); err != nil {
				app.edgexSdk.LoggingClient.Error("error while decoding tag read data: " + err.Error())
			} else {
				app.readCh <- inventory.NewAccessReport(reading.Device, reading.Origin, &report)
			}

		}
	}

	return false, nil
}

func (app *inventoryApp) processReadChannel(wg *sync.WaitGroup) {
	defer wg.Done()
	app.edgexSdk.LoggingClient.Info("starting read channel processing")
	for {
		select {
		case <-app.done:
			app.edgexSdk.LoggingClient.Info("exiting read channel processing")
			return
		case r, ok := <-app.readCh:
			if !ok {
				return
			}

			app.edgexSdk.LoggingClient.Debug("handleRoAccessReport", "deviceName", r.DeviceName, "tagCount", len(r.TagReports))
			e, err := app.processor.ProcessReports(r)
			if err == nil && e != nil && len(e.Params.Data) > 0 {
				app.eventCh <- e
			}
		}
	}
}

func (app *inventoryApp) processEventChannel(wg *sync.WaitGroup) {
	defer wg.Done()
	app.edgexSdk.LoggingClient.Info("starting event channel processing")
	for {
		select {
		case <-app.done:
			app.edgexSdk.LoggingClient.Info("exiting event channel processing")
			return
		// TODO: publish these events somewhere (MQTT, rest, database?)
		case e := <-app.eventCh:
			app.edgexSdk.LoggingClient.Info(fmt.Sprintf("processing event: %+v", e))
			app.pushEventToCoreData(e)
		}
	}
}

func (app *inventoryApp) pushEventToCoreData(event *jsonrpc.InventoryEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		app.edgexSdk.LoggingClient.Error("error marshalling event: " + err.Error())
		return
	}

	if app.edgexSdkContext == nil {
		app.edgexSdk.LoggingClient.Error("unable to push event to core data due to app-functions-sdk context has not been grabbed yet")
		return
	}

	if _, err = app.edgexSdkContext.PushToCoreData(LLRPDeviceService, ResourceInventoryEvent, string(payload)); err != nil {
		app.edgexSdk.LoggingClient.Error("unable to push inventory event to core-data: " + err.Error())
	}
}

func passSettings(settings routes.SettingsHandler, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter,
		r *http.Request) {
		ctx := context.WithValue(r.Context(), routes.SettingsKey, settings)
		handler(w, r.WithContext(ctx))
	}
}

func addRouteErrorHandler(edgexSdk *appsdk.AppFunctionsSDK, err error) {
	if err != nil {
		edgexSdk.LoggingClient.Error("Error adding route: %v", err.Error())
		os.Exit(-1)
	}
}
