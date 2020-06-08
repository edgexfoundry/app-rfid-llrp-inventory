/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"errors"
	"fmt"
	"strings"
)

type testDataset struct {
	tagPro *TagProcessor
	tagReads     []*Gen2Read
	tags         []*Tag
	readTimeOrig int64
	events       []Event
}

func newTestDataset(tagCount int, tagPro *TagProcessor) testDataset {
	ds := testDataset{}
	ds.initialize(tagCount)
	ds.tagPro = tagPro
	return ds
}

func (ds *testDataset) resetEvents() {
	ds.events = make([]Event, 0)
}

// will generate tagread objects but NOT ingest them yet
func (ds *testDataset) initialize(tagCount int) {
	ds.tagReads = make([]*Gen2Read, tagCount)
	ds.tags = make([]*Tag, tagCount)
	ds.readTimeOrig = UnixMilliNow()

	for i := 0; i < tagCount; i++ {
		ds.tagReads[i] = standardReadData(ds.readTimeOrig)
	}

	ds.resetEvents()
}

// update the tag pointers based on actual ingested data
func (ds *testDataset) updateTagRefs() {
	for i, tagRead := range ds.tagReads {
		ds.tags[i] = tagPro.tags[tagRead.Epc]
	}
}

func (ds *testDataset) setRssiAll(rssi int) {
	for _, tagRead := range ds.tagReads {
		tagRead.Rssi = rssi
	}
}

func (ds *testDataset) setLastReadOnAll(timestamp int64) {
	for _, tagRead := range ds.tagReads {
		tagRead.Timestamp = timestamp
	}
}

func (ds *testDataset) readTag(read *Gen2Read, times int) {
	for i := 0; i < times; i++ {
		e := tagPro.ProcessReadData(read)
		switch e.(type) {
		case Arrived:
			ds.events = append(ds.events, e)
		case Moved:
			ds.events = append(ds.events, e)
		}
	}
}

func (ds *testDataset) readAll(devId string, antId int, rssi int, times int) {
	for _, r := range ds.tagReads {
		r.DeviceId = devId
		r.AntennaId = antId
		r.Rssi = rssi
		ds.readTag(r, times)
	}
}

func (ds *testDataset) size() int {
	return len(ds.tagReads)
}

func (ds *testDataset) verifyAll(expectedState State, expectedLocation string) error {
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

func (ds *testDataset) verifyTag(tagIndex int, expectedState State, expectedLocation string) error {
	tag := ds.tags[tagIndex]

	if tag == nil {
		read := ds.tagReads[tagIndex]
		return fmt.Errorf("Expected tag index %d to not be nil! read object: %v\n\tinventory: %#v", tagIndex, read, tagPro)
	}

	if tag.state != expectedState {
		return fmt.Errorf("tag index %d (%s): state %v does not match expected state %v\n\t%#v", tagIndex, tag.Epc, tag.state, expectedState, tag)
	}

	// if expectedRSP is nil, we do not care to validate that field
	if expectedLocation != "" && tag.Location != expectedLocation {
		return fmt.Errorf("tag index %d (%s): location %v does not match expected location %v\n\t%#v", tagIndex, tag.Epc, tag.Location, expectedLocation, tag)
	}

	return nil
}

func (ds *testDataset) verifyStateOf(expectedState State, tagIndex int) error {
	return ds.verifyTag(tagIndex, expectedState, "")
}

func (ds *testDataset) verifyState(tagIndex int, expectedState State) error {
	return ds.verifyTag(tagIndex, expectedState, "")
}

func (ds *testDataset) verifyStateAll(expectedState State) error {
	return ds.verifyAll(expectedState, "")
}

func (ds *testDataset) verifyEventPattern(expectedCount int, expectedEvents ...string) error {
	if expectedCount%len(expectedEvents) != 0 {
		return fmt.Errorf("invalid event pattern specified. pattern length of %d is not evenly divisible by expected event count of %d", len(expectedEvents), expectedCount)
	}

	dataLen := len(ds.events)
	if dataLen != expectedCount {
		return fmt.Errorf("excpected %d %v event pattern to be generated, but %d were generated. events:\n%#v", expectedCount, expectedEvents, dataLen, ds.events)
	}

	for i, item := range ds.events {
		expected := expectedEvents[i%len(expectedEvents)]
		if item.OfType() != string(expected) {
			return fmt.Errorf("excpected %s event but was %s. events:\n%#v", expected, item.OfType(), ds.events)
		}
	}

	return nil
}

func (ds *testDataset) verifyNoEvents() error {
	if len(ds.events) != 0 {
		return fmt.Errorf("excpected no events to be generated, but %d were generated. events:\n%#v", len(ds.events), ds.events)
	}
	return nil
}
