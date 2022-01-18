//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"fmt"
	"testing"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/stretchr/testify/assert"
)

const (
	defaultAntenna = uint16(1)
)

func getTestingLogger() logger.LoggingClient {
	if testing.Verbose() {
		return logger.NewClient("test", "DEBUG")
	}

	return logger.NewMockClient()
}

func TestBasicArrival(t *testing.T) {
	front := nextSensor()

	ds := newTestDataset(NewServiceConfig(), 10)

	events := ds.readAll(t, readParams{
		deviceName: front,
		antenna:    defaultAntenna,
		rssi:       rssiWeak,
		count:      1,
	})

	if err := ds.verifyAll(Present, ds.findAlias(front, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure ALL arrivals WERE generated
	if err := ds.verifyEventPattern(events, ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}
}

func TestTagMoveWeakRssi(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 10)

	back1 := nextSensor()
	back2 := nextSensor()
	back3 := nextSensor()

	// start all tags in the back stock
	events := ds.readAll(t, readParams{
		deviceName: back1,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      1,
	})

	if err := ds.verifyAll(Present, ds.findAlias(back1, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure arrival events generated
	if err := ds.verifyEventPattern(events, ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}

	// move tags to different sensor
	events = ds.readAll(t, readParams{
		deviceName: back2,
		antenna:    defaultAntenna,
		rssi:       rssiStrong,
		count:      4,
	})

	if err := ds.verifyAll(Present, ds.findAlias(back2, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(events, ds.size(), MovedType); err != nil {
		t.Error(err)
	}

	// test that tag stays at new location even with concurrent reads from weaker sensor
	// MOVE back doesn't happen with weak RSSI
	events = ds.readAll(t, readParams{
		deviceName: back3,
		antenna:    defaultAntenna,
		rssi:       rssiWeak,
		count:      1,
	})

	if err := ds.verifyAll(Present, ds.findAlias(back2, defaultAntenna)); err != nil {
		t.Error(err)
	}

	// ensure no events generated
	if err := ds.verifyNoEvents(events); err != nil {
		t.Error(err)
	}
}

func TestMoveAntennaLocation(t *testing.T) {
	initialAntenna := defaultAntenna
	antennaIds := []uint16{2, 4, 33, 15, 99}
	sensor := nextSensor()

	for _, antID := range antennaIds {
		t.Run(fmt.Sprintf("Antenna-%d", antID), func(t *testing.T) {
			ds := newTestDataset(NewServiceConfig(), 1)

			// start all tags at initialAntenna
			events := ds.readAll(t, readParams{
				deviceName: sensor,
				antenna:    initialAntenna,
				rssi:       rssiMin,
				count:      1,
			})

			// ensure arrival events generated
			if err := ds.verifyEventPattern(events, 1, ArrivedType); err != nil {
				t.Error(err)
			}

			epc := ds.epcs[0]
			tag := ds.tp.inventory[epc]
			// move tag to a different antenna port on same sensor
			events = ds.readTag(t, epc, readParams{
				deviceName: sensor,
				antenna:    antID,
				rssi:       rssiStrong,
				count:      4,
			})

			assert.Equalf(t, tag.Location.String(), ds.findAlias(sensor, antID), "tag location was %s, but we expected %s.\n\t%#v", tag.Location.String(), ds.findAlias(sensor, antID), tag)

			// ensure moved events generated
			if err := ds.verifyEventPattern(events, 1, MovedType); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMoveBetweenSensors(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 10)

	back1 := nextSensor()
	back2 := nextSensor()

	// start all tags in the back stock
	events := ds.readAll(t, readParams{
		deviceName: back1,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      1,
	})

	if err := ds.verifyAll(Present, ds.findAlias(back1, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(events, ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}

	// move tag to different sensor
	events = ds.readAll(t, readParams{
		deviceName: back2,
		antenna:    defaultAntenna,
		rssi:       rssiStrong,
		count:      4,
	})

	if err := ds.verifyAll(Present, ds.findAlias(back2, defaultAntenna)); err != nil {
		t.Error(err)
	}

	// ensure moved events generated
	if err := ds.verifyEventPattern(events, ds.size(), MovedType); err != nil {
		t.Error(err)
	}
}

func TestAgeOutTask_RequireDepartedState(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 10)
	sensor := nextSensor()

	// read past ageout threshold
	_ = ds.readAll(t, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		lastSeen:   time.Now().Add(time.Duration(ds.tp.config.ageOutHours) * -3 * time.Hour),
	})

	// make sure all tags are marked as Present and are NOT aged out, because the algorithm
	// should only age out tags that are Departed
	if err := ds.verifyStateAll(Present); err != nil {
		t.Error(err)
	}

	// should not remove any tags
	ds.tp.AgeOut()
	assert.Equalf(t, len(ds.tp.inventory), ds.size(), "expected there to be %d items in the inventory, but there were %d.\ninventory: %#v", ds.size(), len(ds.tp.inventory), ds.tp.inventory)

	// now we will flag the items as departed and run the ageout task again
	_, _ = ds.tp.AggregateDeparted()
	if err := ds.verifyStateAll(Departed); err != nil {
		t.Error(err)
	}
	// this time they should be removed from the inventory
	ds.tp.AgeOut()
	assert.Equalf(t, len(ds.tp.inventory), 0, "expected there to be 0 items in the inventory, but there were %d.\ninventory: %#v", len(ds.tp.inventory), ds.tp.inventory)
}

func TestAgeOutThreshold(t *testing.T) {
	serviceConfig := NewServiceConfig()
	tests := []struct {
		name         string
		lastSeen     time.Time
		state        TagState
		shouldAgeOut bool
	}{
		{
			name:         "Basic age out",
			lastSeen:     time.Now().Add(-1 * time.Duration(2*serviceConfig.AppCustom.AppSettings.AgeOutHours) * time.Hour),
			state:        Departed,
			shouldAgeOut: true,
		},
		{
			name:         "Do not age out",
			lastSeen:     time.Now(),
			state:        Present,
			shouldAgeOut: false,
		},
		{
			name: "Departed but not aged out",
			// 1 hour less than the ageout timeout
			lastSeen:     time.Now().Add(-1 * time.Duration(serviceConfig.AppCustom.AppSettings.AgeOutHours-1) * time.Hour),
			state:        Departed,
			shouldAgeOut: false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ds := newTestDataset(serviceConfig, 5)
			sensor := nextSensor()

			_ = ds.readAll(t, readParams{
				deviceName: sensor,
				antenna:    defaultAntenna,
				lastSeen:   test.lastSeen,
			})

			if err := ds.verifyInventoryCount(ds.size()); err != nil {
				t.Error(err)
			}

			// mark any potential tags as Departed
			_, _ = ds.tp.AggregateDeparted()
			if err := ds.verifyStateAll(test.state); err != nil {
				t.Error(err)
			}

			expectedCount := ds.size()
			if test.shouldAgeOut {
				expectedCount = 0
			}
			// run ageout and check how many tags remain
			ds.tp.AgeOut()
			if err := ds.verifyInventoryCount(expectedCount); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestAggregateDepartedTask(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 10)
	sensor := nextSensor()

	// read past departed threshold
	events := ds.readAll(t, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		count:      10,
		lastSeen:   time.Now().Add(-2 * (time.Duration(ds.tp.config.departedThresholdSeconds) * time.Second)),
	})

	// expect all tags to depart, and their stats to be set to Departed
	events, _ = ds.tp.AggregateDeparted()
	if err := ds.verifyEventPattern(events, ds.size(), DepartedType); err != nil {
		t.Error(err)
	}

	if err := ds.verifyStateAll(Departed); err != nil {
		t.Error(err)
	}

	// read the tags again, this time 1/2 the way between the departed time limit
	// they should all be returned, and generate Arrived events and be Present state
	events = ds.readAll(t, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		count:      10,
		lastSeen:   time.Now().Add(-(time.Duration(ds.tp.config.departedThresholdSeconds) * time.Second) / 2),
	})

	if err := ds.verifyEventPattern(events, ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}
	if err := ds.verifyAll(Present, ds.findAlias(sensor, defaultAntenna)); err != nil {
		t.Error(err)
	}

	// run departed check again, however nothing should depart now because we are
	// within the departed time limit
	events, _ = ds.tp.AggregateDeparted()
	if err := ds.verifyNoEvents(events); err != nil {
		t.Error(err)
	}
}

func TestLastRead_AlwaysIncreasing(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 10)
	sensor := nextSensor()

	current := time.Now()
	_ = ds.readAll(t, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		count:      10,
		lastSeen:   current,
	})

	// make sure the last read is properly set
	if err := ds.verifyLastReadAll(current.UnixNano() / 1e6); err != nil {
		t.Error(err)
	}

	// read all of the tags using the outdated timestamps
	outdated := current.Add(-5 * time.Minute)
	_ = ds.readAll(t, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		count:      10,
		lastSeen:   outdated,
	})

	// make sure the last read was NOT updated, because it was older than current last read
	if err := ds.verifyLastReadAll(current.UnixNano() / 1e6); err != nil {
		t.Error(err)
	}

	// read all of the tags using an even newer timestamp
	next := time.Now()
	_ = ds.readAll(t, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		count:      10,
		lastSeen:   next,
	})

	// make sure the last read WAS updated this time when a newer value was given
	if err := ds.verifyLastReadAll(next.UnixNano() / 1e6); err != nil {
		t.Error(err)
	}
}

func TestAdjustLastReadOnByOrigin(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 2)
	sensor := nextSensor()
	origState := ds.tp.config.adjustLastReadOnByOrigin

	// turn ON the timestamp adjuster
	ds.tp.config.adjustLastReadOnByOrigin = true
	epc0 := ds.epcs[0]
	origin := time.Now()
	lastSeen := origin.Add(-53 * time.Minute)
	_ = ds.readTag(t, epc0, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		lastSeen:   lastSeen,
		origin:     origin,
	})

	// make sure the last read is properly set to the ADJUSTED time, which would be the origin, NOT the
	// lastSeen time
	if err := ds.verifyLastReadOf(epc0, origin.UnixNano()/int64(time.Millisecond)); err != nil {
		t.Error(err)
	}

	// turn OFF the timestamp adjuster
	ds.tp.config.adjustLastReadOnByOrigin = false
	epc1 := ds.epcs[1]
	origin = time.Now()
	lastSeen = origin.Add(-19 * time.Second)
	_ = ds.readTag(t, epc1, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		lastSeen:   lastSeen,
		origin:     origin,
	})

	// make sure the last read is properly set to the ADJUSTED time, which would be the origin, NOT the
	// lastSeen time
	if err := ds.verifyLastReadOf(epc1, lastSeen.UnixNano()/int64(time.Millisecond)); err != nil {
		t.Error(err)
	}

	// put it back to what it was before
	ds.tp.config.adjustLastReadOnByOrigin = origState
}

func TestReaderAntennaAliasDefault(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 0)

	tests := []struct {
		deviceID  string
		antennaID uint16
		expected  string
	}{
		{
			deviceID:  "Reader-3F7DAC",
			antennaID: 1,
			expected:  "Reader-3F7DAC_1",
		},
		{
			deviceID:  "Reader-150000",
			antennaID: 10,
			expected:  "Reader-150000_10",
		},
		{
			deviceID:  "Reader-999999",
			antennaID: 3,
			expected:  "Reader-999999_3",
		},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			alias := ds.findAlias(test.deviceID, test.antennaID)
			assert.Equalf(t, alias, test.expected, "Expected alias of %s, but got %s", test.expected, alias)
		})
	}
}

func TestReaderAntennaAliasExisting(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 0)
	aliasesMap := map[string]string{
		"Reader-3F7DAC_1":  "Freezer",
		"Reader-150000_10": "BackRoom",
		"Reader-999999_3":  "",
	}
	ds.tp.config.aliases = aliasesMap

	tests := []struct {
		deviceID  string
		antennaID uint16
		expected  string
	}{
		{
			deviceID:  "Reader-3F7DAC",
			antennaID: 1,
			expected:  "Freezer",
		},
		{
			deviceID:  "Reader-150000",
			antennaID: 10,
			expected:  "BackRoom",
		},
		{
			deviceID:  "Reader-999999",
			antennaID: 3,
			expected:  "Reader-999999_3",
		},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			alias := ds.findAlias(test.deviceID, test.antennaID)
			assert.Equalf(t, alias, test.expected, "Expected alias of %s, but got %s", test.expected, alias)
		})
	}
}

func TestEventLocationMatchesAlias(t *testing.T) {
	ds := newTestDataset(NewServiceConfig(), 10)
	sensor1 := nextSensor()
	sensor2 := nextSensor()
	alias1 := "Freezer"
	alias2 := "BackRoom"

	// create a time way in the past to ensure tags depart
	origin := time.Now().Add(-99 * time.Hour)

	aliasesMap := map[string]string{
		NewLocation(sensor1, defaultAntenna).String(): alias1,
		NewLocation(sensor2, defaultAntenna).String(): alias2,
	}
	ds.tp.config.aliases = aliasesMap

	// Generate arrived events at alias1
	events := ds.readAll(t, readParams{
		deviceName: sensor1,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      10,
		lastSeen:   origin,
		origin:     origin,
	})
	if err := ds.verifyEventPattern(events, ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}
	// make sure the Location field matches the alias for Arrived events
	for _, event := range events {
		a := event.(ArrivedEvent)
		assert.Equalf(t, a.Location, alias1, "Expected arrived event location to be %s, but was %s", alias1, a.Location)
	}

	// Generate moved events alias1 -> alias2
	events = ds.readAll(t, readParams{
		deviceName: sensor2,
		antenna:    defaultAntenna,
		rssi:       rssiMax,
		count:      10,
		lastSeen:   origin,
		origin:     origin,
	})
	if err := ds.verifyEventPattern(events, ds.size(), MovedType); err != nil {
		t.Error(err)
	}
	// make sure the 2 Location fields match the alias for Moved events
	for _, event := range events {
		m := event.(MovedEvent)
		assert.Equalf(t, m.OldLocation, alias1, "Expected moved event old location to be %s, but was %s", alias1, m.OldLocation)

		assert.Equalf(t, m.NewLocation, alias2, "Expected moved event new location to be %s, but was %s", alias2, m.NewLocation)

	}

	// Generate departed events
	events, _ = ds.tp.AggregateDeparted()
	if err := ds.verifyEventPattern(events, ds.size(), DepartedType); err != nil {
		t.Error(err)
	}
	// make sure the Location field matches the alias for Departed events
	for _, event := range events {
		d := event.(DepartedEvent)
		assert.Equalf(t, d.LastKnownLocation, alias2, "Expected departed event last known location to be %s, but was %s", alias2, d.LastKnownLocation)

	}
}
