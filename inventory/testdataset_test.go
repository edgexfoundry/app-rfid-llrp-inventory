/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"errors"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/helper"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"strings"
)

type testDataset struct {
	tp      *TagProcessor
	lc      logger.LoggingClient
	eventCh chan<- Event

	tagReads     []*llrp.TagReportData
	tags         []*Tag
	readTimeOrig int64
}

func newTestDataset(tp *TagProcessor, eventCh chan<- Event, tagCount int) testDataset {
	ds := testDataset{
		tp:      tp,
		lc:      tp.lc,
		eventCh: eventCh,
	}
	ds.initialize(tagCount)
	return ds
}

func (ds *testDataset) resetEvents() {
	ds.tp.lc.Info("resetEvents() called")
}

// will generate tagread objects but NOT ingest them yet
func (ds *testDataset) initialize(tagCount int) {
	ds.tagReads = make([]*llrp.TagReportData, tagCount)
	ds.tags = make([]*Tag, tagCount)
	ds.readTimeOrig = helper.UnixMilliNow()

	for i := 0; i < tagCount; i++ {
		ds.tagReads[i] = generateReadData(ds.readTimeOrig)
	}

	ds.resetEvents()
}

// update the tag pointers based on actual ingested data
func (ds *testDataset) updateTagRefs() {
	for i, tagRead := range ds.tagReads {
		ds.tags[i] = ds.tp.inventory[tagRead.EPC]
	}
}

func (ds *testDataset) setLastReadOnAll(timestamp int64) {
	for _, tagRead := range ds.tagReads {
		tagRead.LastRead = timestamp
	}
}

func (ds *testDataset) readTag(tagIndex int, deviceName string, antenna int, rssi float64, times int) {
	ds.tagReads[tagIndex].RSSI = rssi
	ds.tagReads[tagIndex].DeviceName = deviceName
	ds.tagReads[tagIndex].Antenna = antenna

	now := helper.UnixMilliNow()
	for i := 0; i < times; i++ {
		ds.tp.process(now, ds.tagReads[tagIndex], ds.eventCh)
	}
}

func (ds *testDataset) readAll(deviceName string, antenna int, rssi float64, times int) {
	for tagIndex := range ds.tagReads {
		ds.readTag(tagIndex, deviceName, antenna, rssi, times)
	}
}

func (ds *testDataset) size() int {
	return len(ds.tagReads)
}

func (ds *testDataset) verifyAll(expectedState TagState, expectedLocation string) error {
	ds.updateTagRefs()

	var errs []string
	for i := range ds.tags {
		if err := ds.verifyTag(i, expectedState, expectedLocation); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (ds *testDataset) verifyTag(tagIndex int, expectedState TagState, expectedLocation string) error {
	tag := ds.tags[tagIndex]

	if tag == nil {
		read := ds.tagReads[tagIndex]
		return fmt.Errorf("Expected tag index %d to not be nil! read object: %v\n\tinventory: %#v", tagIndex, read, ds.tp.inventory)
	}

	if tag.state != expectedState {
		return fmt.Errorf("tag index %d (%s): state %v does not match expected state %v\n\t%#v", tagIndex, tag.EPC, tag.state, expectedState, tag)
	}

	// if expectedLocation is empty string, we do not care to validate that field
	if expectedLocation != "" && tag.Location != expectedLocation {
		return fmt.Errorf("tag index %d (%s): location %v does not match expected location %v\n\t%#v", tagIndex, tag.EPC, tag.Location, expectedLocation, tag)
	}

	return nil
}

func (ds *testDataset) verifyStateOf(expectedState TagState, tagIndex int) error {
	return ds.verifyTag(tagIndex, expectedState, "")
}

func (ds *testDataset) verifyState(tagIndex int, expectedState TagState) error {
	return ds.verifyTag(tagIndex, expectedState, "")
}

func (ds *testDataset) verifyStateAll(expectedState TagState) error {
	return ds.verifyAll(expectedState, "")
}

func (ds *testDataset) verifyEventPattern(expectedCount int, expectedEvents ...EventType) error {
	if expectedCount%len(expectedEvents) != 0 {
		return fmt.Errorf("invalid event pattern specified. pattern length of %d is not evenly divisible by expected event count of %d", len(expectedEvents), expectedCount)
	}

	dataLen := len(ds.inventoryEvent.Params.Data)
	if dataLen != expectedCount {
		return fmt.Errorf("excpected %d %v event pattern to be generated, but %d were generated. events:\n%#v", expectedCount, expectedEvents, dataLen, ds.inventoryEvent.Params.Data)
	}

	for i, item := range ds.inventoryEvent.Params.Data {
		expected := expectedEvents[i%len(expectedEvents)]
		if item.EventType != string(expected) {
			return fmt.Errorf("excpected %s event but was %s. events:\n%#v", expected, item.EventType, ds.inventoryEvent.Params.Data)
		}
	}

	return nil
}

func (ds *testDataset) verifyNoEvents() error {
	if !ds.inventoryEvent.IsEmpty() {
		return fmt.Errorf("excpected no events to be generated, but %d were generated. events:\n%#v", len(ds.inventoryEvent.Params.Data), ds.inventoryEvent.Params.Data)
	}

	return nil
}
