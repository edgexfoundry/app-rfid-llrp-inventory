/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"math"
	"strings"
	"sync"
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

	eventCh chan Event
	events  []Event
	eventMu sync.RWMutex
}

func newTestDataset(lc logger.LoggingClient, tagCount int) *testDataset {
	// buffer the channel enough that we wont ever get blocked
	eventCh := make(chan Event, tagCount)

	ds := testDataset{
		tp:      NewTagProcessor(lc, eventCh),
		lc:      lc,
		epcs:    make([]string, tagCount),
		eventCh: eventCh,
		events:  make([]Event, 0),
	}

	for i := 0; i < tagCount; i++ {
		ds.epcs[i] = nextEPC()
	}

	return &ds
}

func (ds *testDataset) readTag(epc string, deviceName string, antenna int, rssi float64, tm time.Time, count int) {
	rss := llrp.PeakRSSI(rssi)
	ant := llrp.AntennaID(antenna)
	seen := llrp.LastSeenUTC(tm.UnixNano() / int64(time.Microsecond))

	epcBytes, err := hex.DecodeString(epc)
	if err != nil {
		panic(err)
	}
	for i := 0; i < count; i++ {
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

		ds.tp.ProcessReport(r, ReportInfo{
			DeviceName:         deviceName,
			OriginNanos:        tm.UnixNano(),
			offsetMicros:       0,
			referenceTimestamp: tm.UnixNano() / int64(time.Millisecond),
		})
	}
}

func (ds *testDataset) readAll(deviceName string, antenna int, rssi float64, tm time.Time, count int) {
	for _, epc := range ds.epcs {
		ds.readTag(epc, deviceName, antenna, rssi, tm, count)
	}
}

func (ds *testDataset) sniffEvents() {
	ds.eventMu.Lock()
	defer ds.eventMu.Unlock()

	ds.events = make([]Event, 0)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range ds.eventCh {
			ds.events = append(ds.events, e)
		}
	}()

	close(ds.tp.eventCh)
	wg.Wait()
	ds.eventCh = make(chan Event, ds.size())
	ds.tp.eventCh = ds.eventCh
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
	ds.tp.inventoryMu.RLock()
	defer ds.tp.inventoryMu.RUnlock()

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

	ds.eventMu.RLock()
	defer ds.eventMu.RUnlock()

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
	ds.eventMu.RLock()
	defer ds.eventMu.RUnlock()

	if len(ds.events) != 0 {
		return fmt.Errorf("excpected no events to be generated, but %d were generated. events:\n%#v",
			len(ds.events), ds.events)
	}

	return nil
}
