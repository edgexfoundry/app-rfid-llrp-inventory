//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"math"
	"strings"
	"sync/atomic"
	"time"
)

var (
	rssiMin    = float64(-95)
	rssiMax    = float64(-55)
	rssiStrong = rssiMax - math.Floor((rssiMax-rssiMin)/3)
	rssiWeak   = rssiMin + math.Floor((rssiMax-rssiMin)/3)

	tagSerialCounter uint32
	sensorIdCounter  uint32 = 0
)

func nextSensor() string {
	sensorID := atomic.AddUint32(&sensorIdCounter, 1)
	return fmt.Sprintf("Sensor-%02X", sensorID)
}

func nextEPC() string {
	serial := atomic.AddUint32(&tagSerialCounter, 1)
	return fmt.Sprintf("%06x", serial)
}

type testDataset struct {
	tp   *TagProcessor
	lc   logger.LoggingClient
	epcs []string

	events []Event
}

func newTestDataset(lc logger.LoggingClient, tagCount int) *testDataset {
	ds := testDataset{
		tp:     NewTagProcessor(lc, nil),
		lc:     lc,
		epcs:   make([]string, tagCount),
		events: make([]Event, 0),
	}

	for i := 0; i < tagCount; i++ {
		ds.epcs[i] = nextEPC()
	}

	return &ds
}

type readParams struct {
	deviceName string
	antenna    uint16
	rssi       float64
	lastSeen   time.Time
	count      int
	origin     time.Time
}

func (params *readParams) sanitize() {
	if params.lastSeen.Equal(time.Time{}) {
		params.lastSeen = time.Now()
	}
	if params.origin.Equal(time.Time{}) {
		params.origin = params.lastSeen
	}
	if params.count <= 0 {
		params.count = 1
	}
	if params.rssi >= 0 {
		params.rssi = rssiMin
	}
}

// findAlias is a helper method to make the default alias for a device and antenna and lookup any
// associated alias
func (ds *testDataset) findAlias(deviceID string, antID uint16) string {
	return ds.tp.getAlias(makeDefaultAlias(deviceID, antID))
}

func (ds *testDataset) readTag(epc string, params readParams) {
	params.sanitize()

	rss := llrp.PeakRSSI(params.rssi)
	ant := llrp.AntennaID(params.antenna)
	seen := llrp.LastSeenUTC(params.lastSeen.UnixNano() / int64(time.Microsecond))

	epcBytes, err := hex.DecodeString(epc)
	if err != nil {
		panic(err)
	}
	for i := 0; i < params.count; i++ {
		r := &llrp.ROAccessReport{
			TagReportData: []llrp.TagReportData{
				{
					EPC96: llrp.EPC96{
						EPC: epcBytes,
					},
					PeakRSSI:    &rss,
					LastSeenUTC: &seen,
					AntennaID:   &ant,
				},
			},
		}

		events, _ := ds.tp.ProcessReport(r, ReportInfo{
			DeviceName:         params.deviceName,
			OriginNanos:        params.origin.UnixNano(),
			offsetMicros:       0,
			referenceTimestamp: params.origin.UnixNano() / int64(time.Millisecond),
		})

		ds.events = append(ds.events, events...)
	}
}

func (ds *testDataset) readAll(params readParams) {
	for _, epc := range ds.epcs {
		ds.readTag(epc, params)
	}
}

func (ds *testDataset) size() int {
	return len(ds.epcs)
}

func (ds *testDataset) verifyAll(expectedState TagState, expectedLocation string) error {
	var errs []string
	for _, epc := range ds.epcs {
		if err := ds.verifyTag(epc, expectedState, expectedLocation); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (ds *testDataset) verifyTag(epc string, expectedState TagState, expectedLocation string) error {
	tag := ds.tp.inventory[epc]

	if tag == nil {
		return fmt.Errorf("expected tag %s to not be nil!\n\tinventory: %#v", epc, ds.tp.inventory)
	}

	if tag.state != expectedState {
		return fmt.Errorf("tag %s: state %v does not match expected state %v\n\t%#v", epc, tag.state, expectedState, tag)
	}

	// if expectedLocation is empty string, we do not care to validate that field
	if expectedLocation != "" && tag.Location != expectedLocation {
		return fmt.Errorf("tag %s: location %v does not match expected location %v\n\t%#v", epc, tag.Location, expectedLocation, tag)
	}

	return nil
}

func (ds *testDataset) verifyStateOf(epc string, expectedState TagState) error {
	return ds.verifyTag(epc, expectedState, "")
}

func (ds *testDataset) verifyState(epc string, expectedState TagState) error {
	return ds.verifyTag(epc, expectedState, "")
}

func (ds *testDataset) verifyStateAll(expectedState TagState) error {
	return ds.verifyAll(expectedState, "")
}

func (ds *testDataset) verifyEventPattern(expectedCount int, expectedEvents ...EventType) error {
	if expectedCount%len(expectedEvents) != 0 {
		return fmt.Errorf("invalid event pattern specified. pattern length of %d is not evenly divisible by expected event count of %d", len(expectedEvents), expectedCount)
	}

	dataLen := len(ds.events)
	if dataLen != expectedCount {
		return fmt.Errorf("excpected %d %v event pattern to be generated, but %d were generated. events:\n%#v",
			expectedCount, expectedEvents, dataLen, ds.events)
	}

	for i, evt := range ds.events {
		expected := expectedEvents[i%len(expectedEvents)]
		if evt.OfType() != expected {
			return fmt.Errorf("excpected %s event but was %s. events:\n%#v",
				expected, evt.OfType(), ds.events)
		}
	}

	return nil
}

func (ds *testDataset) verifyNoEvents() error {
	if len(ds.events) != 0 {
		return fmt.Errorf("excpected no events to be generated, but %d were generated. events:\n%#v",
			len(ds.events), ds.events)
	}

	return nil
}

func (ds *testDataset) verifyLastReadOf(epc string, lastRead int64) error {
	tag := ds.tp.inventory[epc]

	if tag == nil {
		return fmt.Errorf("expected tag %s to not be nil!\n\tinventory: %#v", epc, ds.tp.inventory)
	}

	if tag.LastRead != lastRead {
		return fmt.Errorf("expected tag %s lastRead to be %d, but was %d", epc, lastRead, tag.LastRead)
	}

	return nil
}

func (ds *testDataset) verifyLastReadAll(lastRead int64) error {
	var errs []string
	for _, epc := range ds.epcs {
		if err := ds.verifyLastReadOf(epc, lastRead); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func (ds *testDataset) verifyInventoryCount(count int) error {
	if len(ds.tp.inventory) != count {
		return fmt.Errorf("expected there to be %d items in the inventory, but there were %d.\ninventory: %#v",
			count, len(ds.tp.inventory), ds.tp.inventory)
	}
	return nil
}
