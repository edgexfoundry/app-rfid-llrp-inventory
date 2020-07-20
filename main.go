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
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/commands/routes"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
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

	ResourceGen2TagRead    = "Gen2TagRead"
	ResourceInventoryEvent = "InventoryEvent"

	// todo: this should probably be configurable
	LLRPDeviceService = "LLRPDeviceService"
)

type inventoryApp struct {
	edgexSdk        *appsdk.AppFunctionsSDK
	edgexSdkContext *appcontext.Context

	processor *inventory.TagProcessor
	readCh    chan inventory.Gen2Read
	eventCh   chan inventory.Event

	done chan interface{}
}

var app inventoryApp

func main() {

	app = inventoryApp{}
	// initialize Edgex App functions SDK
	app.edgexSdk = &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := app.edgexSdk.Initialize(); err != nil {
		app.edgexSdk.LoggingClient.Error(fmt.Sprintf("SDK initialization failed: %v\n", err))
		os.Exit(-1)
	}
	app.done = make(chan interface{})
	app.readCh = make(chan inventory.Gen2Read, readChBuffSz)
	app.eventCh = make(chan inventory.Event, eventChBuffSz)
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

	err = app.edgexSdk.AddRoute("/ping", passSettings(settingsHandlerVar, routes.PingResponse), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/inventory/raw", passSettings(settingsHandlerVar, routes.RawInventory), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/ping", passSettings(settingsHandlerVar, routes.PingResponse), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/command/readers", passSettings(settingsHandlerVar, routes.GetDevicesCommand), http.MethodGet)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/command/readings/{readCommand}", passSettings(settingsHandlerVar, routes.IssueReadCommand), http.MethodPut)
	addRouteErrorHandler(app.edgexSdk, err)

	err = app.edgexSdk.AddRoute("/command/behaviors/{behaviorCommand}", passSettings(settingsHandlerVar, routes.IssueBehaviorCommand), http.MethodPut)
	addRouteErrorHandler(app.edgexSdk, err)

	// the collection of functions to execute every time an event is triggered.
	err = app.edgexSdk.SetFunctionsPipeline(
		app.contextGrabber,
		transforms.NewFilter([]string{ResourceGen2TagRead}).FilterByValueDescriptor,
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
func (app *inventoryApp) contextGrabber(edgexcontext *appcontext.Context, params ...interface{}) (bool, interface{}) {
	if app.edgexSdkContext == nil {
		app.edgexSdkContext = edgexcontext
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

		case ResourceGen2TagRead:
			gen2Read := inventory.Gen2Read{}
			if err := decode(reading.Value, &gen2Read); err == nil {
				app.readCh <- gen2Read
			} else {
				app.edgexSdk.LoggingClient.Error("error while decoding tag read data: " + err.Error())
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
		case r := <-app.readCh:
			app.handleGen2Read(&r)
		}
	}
}

func (app *inventoryApp) handleGen2Read(read *inventory.Gen2Read) {
	app.edgexSdk.LoggingClient.Info(fmt.Sprintf("handleGen2Read from %s", read.DeviceId))
	e := app.processor.ProcessReadData(read)
	switch e.(type) {
	case inventory.Arrived:
		app.eventCh <- e
	case inventory.Moved:
		app.eventCh <- e
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
			app.edgexSdk.LoggingClient.Info(fmt.Sprintf("processing %s event: %+v", e.OfType(), e))
			app.pushEventToCoreData(e)
		}
	}
}

func (app *inventoryApp) pushEventToCoreData(event inventory.Event) {
	payload, err := json.Marshal(event)
	if err != nil {
		app.edgexSdk.LoggingClient.Error("error marshalling event: " + err.Error())
		return
	}

	if app.edgexSdkContext == nil {
		app.edgexSdk.LoggingClient.Error("unable to push event to core data due to app-functions-sdk context has not been grabbed yet")
		return
	}

	if _, err = app.edgexSdkContext.PushToCoreData(LLRPDeviceService, ResourceInventoryEvent+event.OfType(), string(payload)); err != nil {
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

func decode(value string, data interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()

	if err := decoder.Decode(data); err != nil {
		return err
	}

	return nil
}
