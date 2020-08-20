/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"os"
	"testing"
	"time"
)

const (
	defaultAntenna = 1
)

var (
	lc logger.LoggingClient
)

func TestMain(m *testing.M) {
	// todo: when config is implemented again

	//if err := config.InitConfig(); err != nil {
	//	log.Fatal(err)
	//}
	lc = logger.NewClient("test", false, "", "DEBUG")
	os.Exit(m.Run())
}

func TestBasicArrival(t *testing.T) {
	front := nextSensor()

	ds := newTestDataset(lc, 10)

	ds.readAll(readParams{
		deviceName: front,
		antenna:    defaultAntenna,
		rssi:       rssiWeak,
		count:      1,
	})
	ds.sniffEvents()

	if err := ds.verifyAll(Present, GetAntennaAlias(front, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure ALL arrivals WERE generated
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}
}

func TestTagMoveWeakRssi(t *testing.T) {
	ds := newTestDataset(lc, 10)

	back1 := nextSensor()
	back2 := nextSensor()
	back3 := nextSensor()

	// start all tags in the back stock
	ds.readAll(readParams{
		deviceName: back1,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      1,
	})
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back1, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure arrival events generated
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}

	// move tags to different sensor
	ds.readAll(readParams{
		deviceName: back2,
		antenna:    defaultAntenna,
		rssi:       rssiStrong,
		count:      4,
	})
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back2, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), MovedType); err != nil {
		t.Error(err)
	}

	// test that tag stays at new location even with concurrent reads from weaker sensor
	// MOVE back doesn't happen with weak RSSI
	ds.readAll(readParams{
		deviceName: back3,
		antenna:    defaultAntenna,
		rssi:       rssiWeak,
		count:      1,
	})
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back2, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure no events generated
	if err := ds.verifyNoEvents(); err != nil {
		t.Error(err)
	}
}

func TestMoveAntennaLocation(t *testing.T) {
	initialAntenna := defaultAntenna
	antennaIds := []int{2, 4, 33, 15, 99}
	sensor := nextSensor()

	for _, antID := range antennaIds {
		t.Run(fmt.Sprintf("Antenna-%d", antID), func(t *testing.T) {
			ds := newTestDataset(lc, 1)

			// start all tags at initialAntenna
			ds.readAll(readParams{
				deviceName: sensor,
				antenna:    initialAntenna,
				rssi:       rssiMin,
				count:      1,
			})
			ds.sniffEvents()
			// ensure arrival events generated
			if err := ds.verifyEventPattern(1, ArrivedType); err != nil {
				t.Error(err)
			}

			epc := ds.epcs[0]
			tag := ds.tp.inventory[epc]
			// move tag to a different antenna port on same sensor
			ds.readTag(epc, readParams{
				deviceName: sensor,
				antenna:    antID,
				rssi:       rssiStrong,
				count:      4,
			})
			ds.sniffEvents()
			if tag.Location != GetAntennaAlias(sensor, antID) {
				t.Errorf("tag location was %s, but we expected %s.\n\t%#v",
					tag.Location, GetAntennaAlias(sensor, antID), tag)
			}
			// ensure moved events generated
			if err := ds.verifyEventPattern(1, MovedType); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMoveBetweenSensors(t *testing.T) {
	ds := newTestDataset(lc, 10)

	back1 := nextSensor()
	back2 := nextSensor()

	// start all tags in the back stock
	ds.readAll(readParams{
		deviceName: back1,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      1,
	})
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back1, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}

	// move tag to same facility, different sensor
	ds.readAll(readParams{
		deviceName: back2,
		antenna:    defaultAntenna,
		rssi:       rssiStrong,
		count:      4,
	})
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back2, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), MovedType); err != nil {
		t.Error(err)
	}
}

func TestTagProcessor_DoAgeoutTask(t *testing.T) {

}

func TestTagProcessor_DoAggregateDepartedTask(t *testing.T) {
	ds := newTestDataset(lc, 10)
	sensor := nextSensor()

	// read past departed threshold
	ds.readAll(readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      10,
		lastSeen:   time.Now().Add(-2 * (time.Duration(AggregateDepartedThresholdMillis) * time.Millisecond)),
	})
	ds.sniffEvents()

	// expect all tags to depart, and their stats to be set to Departed
	ds.tp.DoAggregateDepartedTask()
	ds.sniffEvents()
	if err := ds.verifyEventPattern(ds.size(), DepartedType); err != nil {
		t.Error(err)
	}
	if err := ds.verifyAll(Departed, ""); err != nil {
		t.Error(err)
	}

	// read the tags again, this time 1/2 the way between the departed time limit
	// they should all be returned, and generate Arrived events and be Present state
	ds.readAll(readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      10,
		lastSeen:   time.Now().Add(-(time.Duration(AggregateDepartedThresholdMillis) * time.Millisecond) / 2),
	})
	ds.sniffEvents()
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}
	if err := ds.verifyAll(Present, GetAntennaAlias(sensor, defaultAntenna)); err != nil {
		t.Error(err)
	}

	// run departed check again, however nothing should depart now because we are
	// within the departed time limit
	ds.tp.DoAggregateDepartedTask()
	ds.sniffEvents()
	if err := ds.verifyNoEvents(); err != nil {
		t.Error(err)
	}
}

func TestLastRead_AlwaysIncreasing(t *testing.T) {
	ds := newTestDataset(lc, 10)
	sensor := nextSensor()

	current := time.Now()
	ds.readAll(readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      10,
		lastSeen:   current,
	})
	ds.sniffEvents()
	// make sure the last read is properly set
	if err := ds.verifyLastReadAll(current.UnixNano() / int64(time.Millisecond)); err != nil {
		t.Error(err)
	}

	// read all of the tags using the outdated timestamps
	outdated := current.Add(-5 * time.Minute)
	ds.readAll(readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      10,
		lastSeen:   outdated,
	})
	ds.sniffEvents()
	// make sure the last read was NOT updated, because it was older than current last read
	if err := ds.verifyLastReadAll(current.UnixNano() / int64(time.Millisecond)); err != nil {
		t.Error(err)
	}

	// read all of the tags using an even newer timestamp
	next := time.Now()
	ds.readAll(readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		rssi:       rssiMin,
		count:      10,
		lastSeen:   next,
	})
	ds.sniffEvents()
	// make sure the last read WAS updated this time when a newer value was given
	if err := ds.verifyLastReadAll(next.UnixNano() / int64(time.Millisecond)); err != nil {
		t.Error(err)
	}

}

func TestAdjustLastReadOnByOrigin(t *testing.T) {
	origState := AdjustLastReadOnByOrigin
	ds := newTestDataset(lc, 2)
	sensor := nextSensor()

	// turn ON the timestamp adjuster
	AdjustLastReadOnByOrigin = true
	epc0 := ds.epcs[0]
	origin := time.Now()
	lastSeen := origin.Add(-53 * time.Minute)
	ds.readTag(epc0, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		lastSeen:   lastSeen,
		origin:     origin,
	})
	ds.sniffEvents()
	// make sure the last read is properly set to the ADJUSTED time, which would be the origin, NOT the
	// lastSeen time
	if err := ds.verifyLastReadOf(epc0, origin.UnixNano()/int64(time.Millisecond)); err != nil {
		t.Error(err)
	}

	// turn OFF the timestamp adjuster
	AdjustLastReadOnByOrigin = false
	epc1 := ds.epcs[1]
	origin = time.Now()
	lastSeen = origin.Add(-19 * time.Second)
	ds.readTag(epc1, readParams{
		deviceName: sensor,
		antenna:    defaultAntenna,
		lastSeen:   lastSeen,
		origin:     origin,
	})
	ds.sniffEvents()
	// make sure the last read is properly set to the ADJUSTED time, which would be the origin, NOT the
	// lastSeen time
	if err := ds.verifyLastReadOf(epc1, lastSeen.UnixNano()/int64(time.Millisecond)); err != nil {
		t.Error(err)
	}

	// put it back to what it was before
	AdjustLastReadOnByOrigin = origState
}
