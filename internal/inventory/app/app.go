//
// Copyright (C) 2020, 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"context"
	"edgexfoundry-holding/rfid-llrp-inventory-service/internal/inventory"
	"edgexfoundry-holding/rfid-llrp-inventory-service/internal/llrp"
	"fmt"
	"github.com/edgexfoundry/app-functions-sdk-go/appsdk"
	"github.com/edgexfoundry/app-functions-sdk-go/pkg/transforms"
	"github.com/edgexfoundry/go-mod-configuration/configuration"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const (
	serviceKey = "rfid-llrp-inventory"

	cacheFolder  = "cache"
	tagCacheFile = "tags.json"
	folderPerm   = 0755 // folders require the execute flag in order to create new files
	filePerm     = 0644
)

type InventoryApp struct {
	edgexSdk     *appsdk.AppFunctionsSDK
	lc           logger.LoggingClient
	devMu        sync.RWMutex
	devService   llrp.DSClient
	defaultGrp   *llrp.ReaderGroup
	snapshotReqs chan snapshotDest
	reports      chan reportData
	configClient configuration.Client
	config       inventory.ConsulConfig
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
	}
}

// Initialize will initialize the AppFunctionsSDK and Logging Client. It also reads the user's
// configuration and sets up the API routes.
func (app *InventoryApp) Initialize() error {
	app.edgexSdk = &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	err := app.edgexSdk.Initialize()
	app.lc = app.edgexSdk.LoggingClient // ensure logging client is assigned before returning
	if err != nil {
		return errors.Wrap(err, "SDK initialization failed")
	}

	app.lc.Info("Starting.")

	appSettings := app.edgexSdk.ApplicationSettings()
	if appSettings == nil {
		return errors.New("missing application settings")
	}
	if app.configClient, err = getConfigClient(); err != nil {
		return errors.Wrap(err, "failed to create config client")
	}

	// todo: switch to using SDK's custom config capability when upgrade to Ireland
	app.config, err = inventory.ParseConsulConfig(app.edgexSdk.LoggingClient, app.edgexSdk.ApplicationSettings())
	if errors.Is(err, inventory.ErrUnexpectedConfigItems) {
		// warn on unexpected config items, but do not exit
		app.lc.Warn(err.Error())
		err = nil
	} else if err != nil {
		return errors.Wrap(err, "config parse error")
	}

	// todo: switch to using EdgeX clients for accessing Core Metadata APIs when upgrade to Ireland
	metadataURI, err := url.Parse(strings.TrimSpace(app.config.ApplicationSettings.MetadataServiceURL))
	if err != nil {
		return errors.Wrap(err, "invalid metadata service URL")
	}
	if metadataURI.Scheme == "" || metadataURI.Host == "" {
		return fmt.Errorf("invalid metadata service URL, endpoint=%s", metadataURI.String())
	}

	devServURI, err := url.Parse(strings.TrimSpace(app.config.ApplicationSettings.DeviceServiceURL))
	if err != nil {
		return errors.Wrap(err, "invalid device service URL")
	}
	if devServURI.Scheme == "" || devServURI.Host == "" {
		return fmt.Errorf("invalid device service URL, endpoint=%s", devServURI.String())
	}

	app.defaultGrp = llrp.NewReaderGroup()
	app.devService = llrp.NewDSClient(&url.URL{
		Scheme: devServURI.Scheme,
		Host:   devServURI.Host,
	}, http.DefaultClient)

	dsName := app.config.ApplicationSettings.DeviceServiceName
	if dsName == "" {
		return errors.New("missing device service name")
	}
	metadataURI.Path = "/api/v1/device/servicename/" + dsName
	deviceNames, err := llrp.GetDevices(metadataURI.String(), http.DefaultClient)
	if err != nil {
		return errors.Wrapf(err, "failed to get existing device names. path=%s", metadataURI.String())
	}
	for _, name := range deviceNames {
		if err = app.defaultGrp.AddReader(app.devService, name); err != nil {
			return fmt.Errorf("failed to setup device %s", name)
		}
	}

	return app.addRoutes()
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
	err := app.edgexSdk.SetFunctionsPipeline(
		transforms.NewFilter([]string{resourceROAccessReport, resourceReaderNotification}).FilterByValueDescriptor,
		app.processEdgeXEvent)
	if err != nil {
		return errors.Wrap(err, "failed to build pipeline")
	}

	if err = app.edgexSdk.MakeItRun(); err != nil {
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
