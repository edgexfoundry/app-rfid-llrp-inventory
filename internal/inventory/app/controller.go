//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"bytes"
	"context"
	"edgexfoundry/app-rfid-llrp-inventory/internal/inventory"
	"edgexfoundry/app-rfid-llrp-inventory/internal/llrp"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/edgexfoundry/app-functions-sdk-go/v2/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/dtos/requests"
	"github.com/pkg/errors"
)

const (
	resourceROAccessReport     = "ROAccessReport"
	resourceReaderNotification = "ReaderEventNotification"
	resourceInventoryEvent     = "InventoryEvent"

	coreDataPostTimeout = 3 * time.Minute
	eventChSz           = 100
)

// processEdgeXEvent is our core processing logic for EdgeX events after they are first
// filtered by the SDK pipeline functions.
//
// Currently it supports two different event types. The first is reader event notifications which
// handles events such as readers being connected and disconnected. The second event type is
// ROAccessReport which is a wrapper around rfid tag read events. These tag readings are sent to
// a channel which processes them as part of the main taskLoop.
func (app *InventoryApp) processEdgeXEvent(_ interfaces.AppFunctionContext, data interface{}) (bool, interface{}) {
	if data == nil {
		return false, errors.New("processEdgeXEvent: No data received")
	}

	event, ok := data.(dtos.Event)
	if !ok {
		return false, errors.New("processEdgeXEvent: didn't receive expected Event type")
	}

	if len(event.Readings) < 1 {
		return false, errors.New("event contains no Readings")
	}

	buff := bytes.Buffer{}
	decoder := json.NewDecoder(&buff)
	decoder.UseNumber()
	decoder.DisallowUnknownFields()

	for i := range event.Readings {
		reading := &event.Readings[i] // Readings is 169 bytes. This avoid the copy.
		switch reading.ResourceName {
		default:
			// this should never happen because it is pre-filtered by the SDK pipeline
			app.lc.Error("Unknown reading name.", "reading", reading.ResourceName)
			continue

		case resourceReaderNotification:
			buff.Reset()
			buff.WriteString(reading.Value)
			notification := &llrp.ReaderEventNotification{}
			if err := decoder.Decode(notification); err != nil {
				app.lc.Error("Failed to decode reader event notification", "error", err.Error())
				continue
			}

			if err := app.handleReaderEvent(event.DeviceName, notification); err != nil {
				app.lc.Error("Failed to handle ReaderEventNotification.",
					"error", err.Error(), "device", event.DeviceName)
			}

		case resourceROAccessReport:
			buff.Reset()
			buff.WriteString(reading.Value)

			report := &llrp.ROAccessReport{}
			if err := decoder.Decode(report); err != nil {
				app.lc.Error("Failed to decode tag report",
					"error", err.Error(), "device", event.DeviceName)
				continue
			}

			if report.TagReportData == nil {
				app.lc.Warn("No tag report data in report.", "device", event.DeviceName)
			} else {
				// pass the tag report data to the reports channel to be processed by our taskLoop
				app.reports <- reportData{report, inventory.NewReportInfo(reading)}
				app.lc.Trace("New ROAccessReport.",
					"device", event.DeviceName, "tags", len(report.TagReportData))
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
func (app *InventoryApp) handleReaderEvent(device string, notification *llrp.ReaderEventNotification) error {
	const connSuccess = llrp.ConnectionAttemptEvent(llrp.ConnSuccess)

	data := notification.ReaderEventNotificationData
	switch {
	case data.ConnectionAttemptEvent != nil && *data.ConnectionAttemptEvent == connSuccess:
		app.lc.Info(fmt.Sprintf("Adding device to default group: %v", device))
		return app.defaultGrp.AddReader(app.devService, device)

	case data.ConnectionCloseEvent != nil:
		app.lc.Info(fmt.Sprintf("Removing device from default group: %v", device))
		app.defaultGrp.RemoveReader(device)
	}

	return nil
}

// requestInventorySnapshot requests that the current inventory snapshot be written to w.
func (app *InventoryApp) requestInventorySnapshot(w io.Writer) error {
	// We send w and a writeErr channel into the inventory execution context
	// and then wait to read a value from the writeErr channel.
	//
	// That context closes writeErr to signal the snapshot is written to w
	// or an error prevented such, and we can send the result back to our caller.
	//
	// This is architected in a way that allows the calling routine to block until the request has
	// been fulfilled by the main taskLoop in a thread-safe manner. This allows callers of the
	// REST API to get a race-free result while also not impacting the performance of the
	// processing logic (ie. thread preemption and mutex locking).
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
func (app *InventoryApp) taskLoop(ctx context.Context) {
	departedCheckSeconds := app.config.AppCustom.AppSettings.DepartedCheckIntervalSeconds
	aggregateDepartedTicker := time.NewTicker(time.Duration(departedCheckSeconds) * time.Second)
	ageoutTicker := time.NewTicker(1 * time.Hour)
	eventCh := make(chan []inventory.Event, eventChSz)

	defer func() {
		aggregateDepartedTicker.Stop()
		ageoutTicker.Stop()
	}()

	// load tag data
	var snapshot []inventory.StaticTag
	snapshotData, err := ioutil.ReadFile(filepath.Join(cacheFolder, tagCacheFile))
	if err != nil {
		app.lc.Warn("Failed to load inventory snapshot.", "error", err.Error())
	} else {
		if err := json.Unmarshal(snapshotData, &snapshot); err != nil {
			app.lc.Warn("Failed to unmarshal inventory snapshot.", "error", err.Error())
		}
	}

	processor := inventory.NewTagProcessor(app.lc, app.config, snapshot)
	if len(snapshot) > 0 {
		app.lc.Info(fmt.Sprintf("Restored %d tags from cache.", len(snapshot)))
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		app.lc.Info("Starting event processor.")
		for events := range eventCh {
			if err := app.pushEventsToCoreData(ctx, events); err != nil {
				app.lc.Error("Failed to push events to CoreData.", "error", err.Error())
			}
		}
		app.lc.Info("Event processor stopped.")
	}()

	app.lc.Info("Starting task loop.")
	for {
		select {
		case <-ctx.Done():
			app.lc.Info("Stopping task loop.")
			close(eventCh)
			app.persistSnapshot(snapshot)
			wg.Wait()
			app.lc.Info("Task loop stopped.")
			return

		case rd := <-app.reports:
			// TODO: we should refactor the ReaderGroup/TagReader
			//   to unite its tag processing with the TagProcessor code;
			//   the biggest goal is to perform only a single pass on the TagReportData.
			//   Secondarily, it would allow us to eliminate the ReaderGroup mutex.
			if !app.defaultGrp.ProcessTagReport(rd.info.DeviceName, rd.report.TagReportData) {
				// This can only happen if the device didn't exist when we started,
				// and we never got a Connection message for it.
				app.lc.Error("Tag Report for unknown device.", "device", rd.info.DeviceName)
			}

			events, updatedSnapshot := processor.ProcessReport(rd.report, rd.info)
			if updatedSnapshot != nil {
				snapshot = updatedSnapshot // always update the snapshot if available
			}
			if len(events) > 0 {
				app.persistSnapshot(snapshot) // only persist when there are inventory events
				eventCh <- events
			}

		case t := <-aggregateDepartedTicker.C:
			app.lc.Debug("Running AggregateDeparted.", "time", fmt.Sprintf("%v", t))

			if events, updatedSnapshot := processor.AggregateDeparted(); len(events) > 0 {
				if updatedSnapshot != nil { // should always be true if there are events
					snapshot = updatedSnapshot
					app.persistSnapshot(snapshot)
				}
				eventCh <- events
			}

		case t := <-ageoutTicker.C:
			app.lc.Debug("Running AgeOut.", "time", fmt.Sprintf("%v", t))
			if _, updatedSnapshot := processor.AgeOut(); updatedSnapshot != nil {
				snapshot = updatedSnapshot
				app.persistSnapshot(snapshot)
			}

		case rawConfig := <-app.confUpdateCh:
			newConfig, ok := rawConfig.(*inventory.CustomConfig)
			if !ok {
				app.lc.Warn("Unable to decode configuration from consul.", "raw", fmt.Sprintf("%#v", rawConfig))
				continue
			}

			if err := newConfig.AppSettings.Validate(); err != nil {
				app.lc.Error("Invalid Consul configuration.", "error", err.Error())
				continue
			}

			app.lc.Info("Configuration updated from consul.")
			app.lc.Debug("New consul config.", "config", fmt.Sprintf("%+v", newConfig))
			processor.UpdateConfig(*newConfig)

			// check if we need to change the ticker interval
			if departedCheckSeconds != newConfig.AppSettings.DepartedCheckIntervalSeconds {
				aggregateDepartedTicker.Stop()
				departedCheckSeconds = newConfig.AppSettings.DepartedCheckIntervalSeconds
				aggregateDepartedTicker = time.NewTicker(time.Duration(departedCheckSeconds) * time.Second)
				app.lc.Info(fmt.Sprintf("Changing aggregate departed check interval to %d seconds.", departedCheckSeconds))
			}

		case req := <-app.snapshotReqs:
			data, err := json.Marshal(snapshot)
			if err == nil {
				_, err = req.w.Write(data) // only write if there was no error already
			}
			req.result <- err
		}
	}
}

func (app *InventoryApp) persistSnapshot(snapshot []inventory.StaticTag) {
	app.lc.Debug("Persisting inventory snapshot.")
	data, err := json.Marshal(snapshot)
	if err != nil {
		app.lc.Warn("Failed to marshal inventory snapshot.", "error", err.Error())
		return
	}

	if err := ioutil.WriteFile(filepath.Join(cacheFolder, tagCacheFile), data, filePerm); err != nil {
		app.lc.Warn("Failed to persist inventory snapshot.", "error", err.Error())
		return
	}
	app.lc.Info("Persisted inventory snapshot.", "tags", len(snapshot))
}

// setDefaultBehavior sets the behavior associated with the default device group.
func (app *InventoryApp) setDefaultBehavior(b llrp.Behavior) error {
	app.devMu.Lock()
	err := app.defaultGrp.SetBehavior(app.devService, b)
	app.devMu.Unlock()
	return err
}

// pushEventsToCoreData will send one or more Inventory Events as a single EdgeX Event with
// an EdgeX Reading for each Inventory Event
func (app *InventoryApp) pushEventsToCoreData(ctx context.Context, events []inventory.Event) error {
	// These events are generated by the app-service itself, so we are using serviceKey
	// for the profile, device, and source names.
	edgeXEvent := dtos.NewEvent(serviceKey, serviceKey, serviceKey)

	var errs []error
	for _, event := range events {
		payload, err := json.Marshal(event)
		if err != nil {
			errs = append(errs, errors.Wrap(err, "error marshalling event"))
			continue
		}

		resourceName := resourceInventoryEvent + string(event.OfType())
		app.service.LoggingClient().Info("Sending Inventory Event.",
			"type", resourceName, "payload", string(payload))

		err = edgeXEvent.AddSimpleReading(resourceName, common.ValueTypeString, string(payload))
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "error creating reading for %s", resourceName))
			continue
		}

	}

	ctx, cancel := context.WithTimeout(ctx, coreDataPostTimeout)
	defer cancel()

	if _, err := app.service.EventClient().Add(ctx, requests.NewAddEventRequest(edgeXEvent)); err != nil {
		errs = append(errs, errors.Wrap(err, "unable to push inventory event(s) to core-data"))
	}

	if errs != nil {
		return llrp.MultiErr(errs)
	}
	return nil
}
