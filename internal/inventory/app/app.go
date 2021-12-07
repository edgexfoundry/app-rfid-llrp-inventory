//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"edgexfoundry/app-rfid-llrp-inventory/internal/inventory"
	"edgexfoundry/app-rfid-llrp-inventory/internal/llrp"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg"
	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/pkg/errors"
)

const (
	serviceKey = "app-rfid-llrp-inventory"

	cacheFolder  = "cache"
	tagCacheFile = "tags.json"
	folderPerm   = 0755 // folders require the execute flag in order to create new files
	filePerm     = 0644
)

type InventoryApp struct {
	service      interfaces.ApplicationService
	lc           logger.LoggingClient
	devMu        sync.RWMutex
	devService   llrp.DSClient
	defaultGrp   *llrp.ReaderGroup
	snapshotReqs chan snapshotDest
	reports      chan reportData
	config       inventory.ServiceConfig
	confUpdateCh chan interface{}
}

type reportData struct {
	report *llrp.ROAccessReport
	info   inventory.ReportInfo
}

type snapshotDest struct {
	w      io.Writer
	result chan error
}

func NewInventoryApp() *InventoryApp {
	return &InventoryApp{
		snapshotReqs: make(chan snapshotDest),
		reports:      make(chan reportData),
		confUpdateCh: make(chan interface{}),
	}
}

// Initialize will initialize the AppFunctionsSDK and Logging Client. It also reads the user's
// configuration and sets up the API routes.
func (app *InventoryApp) Initialize() error {
	var ok bool
	var err error

	app.service, ok = pkg.NewAppService(serviceKey)
	if !ok {
		return errors.New("Failed to create application service")
	}

	app.lc = app.service.LoggingClient() // ensure logging client is assigned before returning

	app.lc.Info("Starting.")

	if err = app.service.LoadCustomConfig(&app.config, aliasesConfigKey); err != nil {
		return errors.Wrap(err, "Failed to load custom configuration")
	}

	if err = app.config.AppCustom.AppSettings.Validate(); err != nil {
		return errors.Wrap(err, "Fail to validate custom config")
	}

	if err = app.service.ListenForCustomConfigChanges(&app.config.AppCustom, "AppCustom", app.processConfigUpdates); err != nil {
		return errors.Wrap(err, "Listen for custom changes Failed")
	}

	app.defaultGrp = llrp.NewReaderGroup()
	app.devService = llrp.NewDSClient(app.service.CommandClient(), app.lc)

	dsName := app.config.AppCustom.AppSettings.DeviceServiceName
	if dsName == "" {
		return errors.New("missing device service name")
	}

	devices, err := llrp.GetDevices(app.service.DeviceClient(), dsName)
	if err != nil {
		return errors.Wrapf(err, "failed to get existing device names for device service name %s", dsName)
	}

	app.lc.Debugf("Found %d devices", len(devices))
	for _, device := range devices {
		app.lc.Debugf("Attempting to add Reader for device '%s'", device.Name)
		if err = app.defaultGrp.AddReader(app.devService, device.Name); err != nil {
			app.lc.Errorf("Failed to setup device %s: %s", device.Name, err.Error())
		}
	}

	return app.addRoutes()
}

func (app *InventoryApp) processConfigUpdates(rawWritableConfig interface{}) {
	app.confUpdateCh <- rawWritableConfig
}

// RunUntilCancelled sets up the function pipeline and runs it. This function will not return
// until the function pipeline is complete unless an error occurred running it.
func (app *InventoryApp) RunUntilCancelled() error {

	if err := os.MkdirAll(cacheFolder, folderPerm); err != nil {
		app.lc.Error("Failed to create cache directory.", "directory", cacheFolder, "error", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.taskLoop(ctx)
		app.lc.Info("Task loop has exited.")
	}()

	// We are doing this because of an issue with running app-functions-sdk inside
	// of docker-compose where something is hanging and not relinquishing control
	// back to our code.
	//
	// Note that this code does not in any way attempt to "fix" the deadlock issue,
	// but instead provides our code a way to cleanup and persist the data safely
	// when the process is exiting.
	//
	// see: https://github.com/edgexfoundry/app-functions-sdk-go/issues/500
	go func() {
		signals := make(chan os.Signal)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		s := <-signals

		app.lc.Info(fmt.Sprintf("Received '%s' signal from OS.", s.String()))
		cancel() // signal the taskLoop to finish
	}()

	// Subscribe to events.
	err := app.service.SetFunctionsPipeline(
		app.processEdgeXEvent)
	if err != nil {
		return errors.Wrap(err, "failed to build pipeline")
	}

	if err = app.service.MakeItRun(); err != nil {
		return errors.Wrap(err, "failed to run pipeline")
	}

	// let task loop complete
	wg.Wait()
	app.lc.Info("Exiting.")

	return nil
}

func (app *InventoryApp) LoggingClient() logger.LoggingClient {
	return app.lc
}
