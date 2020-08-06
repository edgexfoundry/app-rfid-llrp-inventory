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
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"net/http"
	"os"
	"strings"
)

const (
	serviceKey = "rfid-inventory"

	ResourceGen2TagRead    = "Gen2TagRead"
	ResourceInventoryEvent = "InventoryEvent"

	// CoreCommandPUTDevice app settings
	CoreCommandPUTDevice = "CoreCommandPUTDevice"
	// CoreCommandGETDevices app settings
	CoreCommandGETDevices = "CoreCommandGETDevices"
)

func main() {

	// initialize EdgeX App functions SDK
	edgexSdk := &appsdk.AppFunctionsSDK{ServiceKey: serviceKey}
	if err := edgexSdk.Initialize(); err != nil {
		panic(fmt.Sprintf("SDK initialization failed: %v\n", err))
	}

	lgr := edgexSdk.LoggingClient

	lgr.Info("Starting.")

	appSettings := edgexSdk.ApplicationSettings()
	if appSettings == nil {
		lgr.Error("No application settings found.")
		os.Exit(1)
	}

	apiBase := strings.TrimSpace(appSettings[CoreCommandPUTDevice])
	getDevURL := strings.TrimSpace(appSettings[CoreCommandGETDevices])
	if apiBase == "" || getDevURL == "" {
		lgr.Error("Missing endpoint configuration.")
		os.Exit(1)
	}

	tagProc := inventory.NewTagProcessor(lgr)

	for _, rte := range []struct {
		path, method string
		f            http.HandlerFunc
	}{
		{"/", http.MethodGet, routes.Index},
		{"/api/v1/inventory/raw", http.MethodGet, routes.RawInventory(lgr, tagProc)},
		{"/api/v1/command/reading/start", http.MethodPut,
			routes.StartReaders(lgr, apiBase, "StartReading", getDevURL)},
		{"/api/v1/command/behaviors/{behaviorCommand}", http.MethodPut,
			routes.SetBehaviors()},
	} {
		if err := edgexSdk.AddRoute(rte.path, rte.f, rte.path); err != nil {
			lgr.Error("Failed to add route.", "error", err.Error(),
				"path", rte.path, "method", rte.method)
			os.Exit(1)
		}
	}

	// Use the functions pipeline to subscribe to events,
	// but just process them directly so we don't have to
	// sacrifice compile-time type checking,
	// eat the efficiency costs of run-time type checking,
	// or iterate over the same list of Readings multiple times.
	if err := edgexSdk.SetFunctionsPipeline(newEventHandler(tagProc)); err != nil {
		edgexSdk.LoggingClient.Error("Failed to build pipeline.", "error", err.Error())
		os.Exit(1)
	}

	if err := edgexSdk.MakeItRun(); err != nil {
		edgexSdk.LoggingClient.Error("Failed to MakeItRun.", "error", err.Error())
		os.Exit(1)
	}

	// TODO: subscribe to notifications about readers connect/disconnect.
	//   Upon connection, set Reader config/ROSpecs/AccessSpecs.
	//   Check if Supports Events and Report Holding & send messages as needed.
}

func newEventHandler(lgr logger.LoggingClient, tagPro *inventory.TagProcessor) appcontext.AppFunction {
	return func(edgeXCtx *appcontext.Context, params ...interface{}) (bool, interface{}) {
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
			if reading.Name != ResourceGen2TagRead {
				continue
			}

			r.Reset()
			r.WriteString(reading.Value)

			gen2Read := inventory.Gen2Read{}
			if err := decoder.Decode(&gen2Read); err != nil {
				lgr.Error("Failed to decode tag read", "error", err.Error())
				continue
			}

			lgr.Debug("Decoded tag data.", "tagData", gen2Read)
			if e := tagPro.ProcessReadData(&gen2Read); e != nil {
				lgr.Debug("Processing event.", "eventType", e.OfType(),
					"event", fmt.Sprintf("%+v", e))

				payload, err := json.Marshal(e)
				if err != nil {
					lgr.Error("error marshalling event: " + err.Error())
					lgr.Error("Failed to marshal output event.",
						"eventType", e.OfType(), "error", err.Error())
					continue
				}

				eventName := ResourceInventoryEvent + e.OfType()
				if _, err := edgeXCtx.PushToCoreData(event.Device, eventName, string(payload)); err != nil {
					lgr.Error("Failed to push inventory event to core-data.", "error", err.Error())
					continue
				}
			}
		}

		return false, nil
	}
}
