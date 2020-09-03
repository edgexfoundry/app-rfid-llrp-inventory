/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/edgexfoundry/app-functions-sdk-go/pkg/transforms"
	"github.com/edgexfoundry/go-mod-bootstrap/bootstrap/flags"
	"github.com/edgexfoundry/go-mod-configuration/configuration"
	"github.com/edgexfoundry/go-mod-configuration/pkg/types"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/routes"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	serviceKey      = "rfid-inventory"
	eventChBuffSz   = 10
	eventDeviceName = "rfid-inventory"

	BaseConsulPath = "edgex/appservices/1.0/"

	ResourceROAccessReport = "ROAccessReport"

	ResourceInventoryEventArrived  = "InventoryEventArrived"
	ResourceInventoryEventMoved    = "InventoryEventMoved"
	ResourceInventoryEventDeparted = "InventoryEventDeparted"
)

type inventoryApp struct {
	edgexSdk        *appsdk.AppFunctionsSDK
	edgexSdkContext *appcontext.Context

	processor *inventory.TagProcessor
	eventCh   chan inventory.Event

	config   *consulConfig
	configMu sync.RWMutex

	done chan struct{}
}

type consulConfig struct {
	Aliases map[string]string
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
	app.eventCh = make(chan inventory.Event, eventChBuffSz)
	app.config = &consulConfig{
		Aliases: make(map[string]string),
	}
	app.processor = inventory.NewTagProcessor(app.edgexSdk.LoggingClient, app.eventCh)
	if err := app.processor.Restore(inventory.TagCacheFile); err != nil {
		app.edgexSdk.LoggingClient.Warn("An issue occurred restoring tag inventory from cache.",
			"error", err)
	}
	app.edgexSdk.LoggingClient.Info("Running")

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

	err = app.edgexSdk.AddRoute("/inventory/snapshot",
		func(writer http.ResponseWriter, request *http.Request) {
			routes.RawInventory(app.edgexSdkContext.LoggingClient, writer, request, app.processor)
		}, http.MethodGet)
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
	go func() {
		defer wg.Done()
		app.processEventChannel()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		app.processScheduledTasks()
	}()

	if err := app.watchForConfigChanges(); err != nil {
		app.edgexSdk.LoggingClient.Warn("Unable to watch for consul configuration changes.", "error", err)
	}

	// HACK: We are doing this because of an issue with running app-fn-sdk inside
	// of docker-compose where something is hanging and not relinquishing control
	// back to our code.
	go func() {
		signals := make(chan os.Signal)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals

		if err := app.processor.Persist(inventory.TagCacheFile); err != nil {
			app.edgexSdk.LoggingClient.Error("An error occurred persisting tag inventory to cache.",
				"error", err)
		}

		app.edgexSdk.LoggingClient.Info("waiting for channels to finish")
		close(app.done)
	}()

	// tell SDK to "start" and begin listening for events to trigger the pipeline.
	err = app.edgexSdk.MakeItRun()
	if err != nil {
		app.edgexSdk.LoggingClient.Error("MakeItRun returned error: ", err.Error())
		os.Exit(-1)
	}

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
				continue
			}

			app.edgexSdk.LoggingClient.Debug("handleRoAccessReport", "deviceName", reading.Device, "tagCount", len(report.TagReportData))
			app.processor.ProcessReport(&report, inventory.NewReportInfo(reading))
		}
	}

	return false, nil
}

// processScheduledTasks is an infinite loop that processes timer tickers which are basically
// a way to run code on a scheduled interval in golang
func (app *inventoryApp) processScheduledTasks() {
	aggregateDepartedTicker := time.NewTicker(time.Duration(inventory.DepartedCheckIntervalSeconds) * time.Second)
	defer aggregateDepartedTicker.Stop()

	ageoutTicker := time.NewTicker(1 * time.Hour)
	defer ageoutTicker.Stop()

	for {
		select {
		case <-app.done:
			app.edgexSdk.LoggingClient.Info("done called. stopping scheduled tasks")
			return

		case t, ok := <-aggregateDepartedTicker.C:
			if !ok {
				return
			}
			app.edgexSdk.LoggingClient.Debug(fmt.Sprintf("RunAggregateDepartedTask: %v", t))
			app.processor.RunAggregateDepartedTask()

		case t, ok := <-ageoutTicker.C:
			if !ok {
				return
			}
			app.edgexSdkContext.LoggingClient.Debug(fmt.Sprintf("RunAgeOutTask: %v", t))
			app.processor.RunAgeOutTask()
		}
	}
}

func (app *inventoryApp) processEventChannel() {
	app.edgexSdk.LoggingClient.Info("starting event channel processing")
	for {
		select {
		case <-app.done:
			app.edgexSdk.LoggingClient.Info("exiting event channel processing")
			return
		case e, ok := <-app.eventCh:
			if !ok {
				return
			}

			app.edgexSdk.LoggingClient.Info(fmt.Sprintf("processing %s event: %+v", e.OfType(), e))
			if err := app.pushEventToCoreData(e); err != nil {
				app.edgexSdk.LoggingClient.Error(err.Error())
			}
			if err := app.processor.Persist(inventory.TagCacheFile); err != nil {
				app.edgexSdk.LoggingClient.Warn("There was an issue persisting the data", "error", err.Error())
			}
		}
	}
}

func (app *inventoryApp) pushEventToCoreData(event inventory.Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "error marshalling event")
	}

	if app.edgexSdkContext == nil {
		return errors.New("unable to push event to core data due to app-functions-sdk context has not been grabbed yet")
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
		return errors.New("unknown event type!")
	}

	if _, err = app.edgexSdkContext.PushToCoreData(eventDeviceName, resource, string(payload)); err != nil {
		return errors.Wrap(err, "unable to push inventory event to core-data")
	}

	return err
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

// watchForConfigChanges watches for some configuration changes in EdgeX Consul and dynamically updates the application with
// the new changes and also if the application restarts loads the existing config values from Consul
func (app *inventoryApp) watchForConfigChanges() error {
	sdkFlags := flags.New()
	sdkFlags.Parse(os.Args[1:])
	cpUrl, err := url.Parse(sdkFlags.ConfigProviderUrl())
	if err != nil {
		return err
	}

	cpPort := 8500
	port := cpUrl.Port()
	if port != "" {
		cpPort, err = strconv.Atoi(port)
		if err != nil {
			app.edgexSdk.LoggingClient.Error("Error with edgex configuration provider url port: %v", err.Error())
			cpPort = 8500
		}
	}

	configClient, err := configuration.NewConfigurationClient(types.ServiceConfig{
		Host:     cpUrl.Hostname(),
		Port:     cpPort,
		BasePath: BaseConsulPath,
		Type:     cpUrl.Scheme,
	})
	if err != nil {
		return err
	}

	go func() {
		errorStream := make(chan error)
		defer close(errorStream)

		updateStream := make(chan interface{})
		defer close(updateStream)

		configClient.WatchForChanges(updateStream, errorStream, app.config, "/"+serviceKey)
		app.edgexSdk.LoggingClient.Info("Watching for consul configuration changes...")

		for {
			select {
			case <-app.done:
				return

			case ex := <-errorStream:
				app.edgexSdk.LoggingClient.Error(ex.Error())

			case rawConfig, ok := <-updateStream:
				if !ok {
					return
				}

				app.edgexSdk.LoggingClient.Debug(fmt.Sprintf("Raw configuration from Consul: %+v", rawConfig))

				newConfig, ok := rawConfig.(*consulConfig)
				if ok {
					app.edgexSdk.LoggingClient.Info("Configuration from Consul received")
					app.edgexSdk.LoggingClient.Debug(fmt.Sprintf("Configuration from Consul: %#v", newConfig))
					app.processor.SetAliases(newConfig.Aliases)

					app.configMu.Lock()
					app.config = newConfig
					app.configMu.Unlock()
				} else {
					app.edgexSdk.LoggingClient.Warn("Unable to decode configuration from Consul")
				}
			}
		}
	}()
	return nil
}
