/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"errors"
	"fmt"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/sensor"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/jsonrpc"
	"github.com/intel/rsp-sw-toolkit-im-suite-utilities/helper"
	"github.com/sirupsen/logrus"
	"strings"
)

type testDataset struct {
	tagReads       []*jsonrpc.TagRead
	tags           []*Tag
	readTimeOrig   int64
	inventoryEvent *jsonrpc.InventoryEvent
}

func newTestDataset(tagCount int) testDataset {
	ds := testDataset{}
	ds.initialize(tagCount)
	return ds
}

func (ds *testDataset) resetEvents() {
	ds.inventoryEvent = jsonrpc.NewInventoryEvent()
	logrus.Info("resetEvents() called")
}

// will generate tagread objects but NOT ingest them yet
func (ds *testDataset) initialize(tagCount int) {
	ds.tagReads = make([]*jsonrpc.TagRead, tagCount)
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
		ds.tags[i] = inventory[tagRead.Epc]
	}
}

func (ds *testDataset) setRssi(tagIndex int, rssi int) {
	ds.tagReads[tagIndex].Rssi = rssi
}

func (ds *testDataset) setRssiAll(rssi int) {
	for _, tagRead := range ds.tagReads {
		tagRead.Rssi = rssi
	}
}

func (ds *testDataset) setLastReadOnAll(timestamp int64) {
	for _, tagRead := range ds.tagReads {
		tagRead.LastReadOn = timestamp
	}
}

func (ds *testDataset) readTag(tagIndex int, rsp *sensor.RSP, rssi int, times int) {
	ds.setRssi(tagIndex, rssi)

	for i := 0; i < times; i++ {
		processReadData(helper.UnixMilliNow(), ds.inventoryEvent, ds.tagReads[tagIndex], rsp)
	}
}

func (ds *testDataset) readAll(rsp *sensor.RSP, rssi int, times int) {
	for tagIndex := range ds.tagReads {
		ds.readTag(tagIndex, rsp, rssi, times)
	}
}

func (ds *testDataset) size() int {
	return len(ds.tagReads)
}

func (ds *testDataset) verifyAll(expectedState TagState, expectedRSP *sensor.RSP) error {
	ds.updateTagRefs()

	var errs []string
	for i := range ds.tags {
		if err := ds.verifyTag(i, expectedState, expectedRSP); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (ds *testDataset) verifyTag(tagIndex int, expectedState TagState, expectedRSP *sensor.RSP) error {
	tag := ds.tags[tagIndex]

	if tag == nil {
		read := ds.tagReads[tagIndex]
		return fmt.Errorf("Expected tag index %d to not be nil! read object: %v\n\tinventory: %#v", tagIndex, read, inventory)
	}

	if tag.state != expectedState {
		return fmt.Errorf("tag index %d (%s): state %v does not match expected state %v\n\t%#v", tagIndex, tag.EPC, tag.state, expectedState, tag)
	}

	// if expectedRSP is nil, we do not care to validate that field
	if expectedRSP != nil && tag.Location != expectedRSP.AntennaAlias(0) {
		return fmt.Errorf("tag index %d (%s): location %v does not match expected location %v\n\t%#v", tagIndex, tag.EPC, tag.Location, expectedRSP.AntennaAlias(0), tag)
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

func (ds *testDataset) verifyEventPattern(expectedCount int, expectedEvents ...Event) error {
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
