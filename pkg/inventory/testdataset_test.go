/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/helper"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/sensor"
	"strings"
)

type testDataset struct {
	tagReads       []*TagReport
	tags           []*Tag
	readTimeOrig   int64
	inventoryEvent *jsonrpc.InventoryEvent
	tp *TagProcessor
}

func newTestDataset(tp *TagProcessor, tagCount int) testDataset {
	ds := testDataset{
		tp: tp,
	}
	ds.initialize(tagCount)
	return ds
}

func (ds *testDataset) resetEvents() {
	ds.inventoryEvent = jsonrpc.NewInventoryEvent()
	logrus.Info("resetEvents() called")
}

// will generate tagread objects but NOT ingest them yet
func (ds *testDataset) initialize(tagCount int) {
	ds.tagReads = make([]*TagReport, tagCount)
	ds.tags = make([]*Tag, tagCount)
	ds.readTimeOrig = helper.UnixMilliNow()

	for i := 0; i < tagCount; i++ {
		ds.tagReads[i] = generateReadData(ds.readTimeOrig, 1)
	}

	ds.resetEvents()
}

// update the tag pointers based on actual ingested data
func (ds *testDataset) updateTagRefs() {
	for i, tagRead := range ds.tagReads {
		ds.tags[i] = ds.tp.inventory[tagRead.EPC()]
	}
}

func (ds *testDataset) setRssi(tagIndex int, rssi int) {
	v := llrp.PeakRSSI(rssi)
	ds.tagReads[tagIndex].PeakRSSI = &v
}

func (ds *testDataset) setRssiAll(rssi int) {
	v := llrp.PeakRSSI(rssi)
	for _, tagRead := range ds.tagReads {
		tagRead.PeakRSSI = &v
	}
}

func (ds *testDataset) setLastReadOnAll(timestamp int64) {
	ts := llrp.LastSeenUTC(timestamp)
	for _, tagRead := range ds.tagReads {
		tagRead.LastSeenUTC = &ts
	}
}

func (ds *testDataset) readTag(tagIndex int, s *sensor.Sensor, rssi int, times int) {
	ds.setRssi(tagIndex, rssi)

	for i := 0; i < times; i++ {
		ds.tp.process(ds.inventoryEvent, ds.tagReads[tagIndex], s)
	}
}

func (ds *testDataset) readAll(s *sensor.Sensor, rssi int, times int) {
	for tagIndex := range ds.tagReads {
		ds.readTag(tagIndex, s, rssi, times)
	}
}

func (ds *testDataset) size() int {
	return len(ds.tagReads)
}

func (ds *testDataset) verifyAll(expectedState TagState, expecteds *sensor.Sensor) error {
	ds.updateTagRefs()

	var errs []string
	for i := range ds.tags {
		if err := ds.verifyTag(i, expectedState, expecteds); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (ds *testDataset) verifyTag(tagIndex int, expectedState TagState, expecteds *sensor.Sensor) error {
	tag := ds.tags[tagIndex]

	if tag == nil {
		read := ds.tagReads[tagIndex]
		return fmt.Errorf("Expected tag index %d to not be nil! read object: %v\n\tinventory: %#v", tagIndex, read, ds.tp.inventory)
	}

	if tag.state != expectedState {
		return fmt.Errorf("tag index %d (%s): state %v does not match expected state %v\n\t%#v", tagIndex, tag.EPC, tag.state, expectedState, tag)
	}

	// if expecteds is nil, we do not care to validate that field
	if expecteds != nil && tag.Location != expecteds.AntennaAlias(0) {
		return fmt.Errorf("tag index %d (%s): location %v does not match expected location %v\n\t%#v", tagIndex, tag.EPC, tag.Location, expecteds.AntennaAlias(0), tag)
	}

	return nil
}

func (ds *testDataset) verifyStateOf(expectedState TagState, tagIndex int) error {
	return ds.verifyTag(tagIndex, expectedState, nil)
}

func (ds *testDataset) verifyState(tagIndex int, expectedState TagState) error {
	return ds.verifyTag(tagIndex, expectedState, nil)
}

func (ds *testDataset) verifyStateAll(expectedState TagState) error {
	return ds.verifyAll(expectedState, nil)
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
