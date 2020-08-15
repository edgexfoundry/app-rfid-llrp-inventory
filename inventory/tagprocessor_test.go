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

//
//func TestPosDoesNotGenerateArrival(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(lc, 10)
//
//	front := nextSensor()
//	posSensor := nextSensor()
//
//	ds.readAll(posSensor, rssiMin, 1)
//	ds.updateTagRefs()
//	if err := ds.verifyAll(Unknown, sensor.GetAntennaAlias(posSensor, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	// read a few more times, we still do not want to arrive
//	ds.readAll(posSensor, rssiMin, 4)
//	if err := ds.verifyAll(Unknown, sensor.GetAntennaAlias(posSensor, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	ds.readAll(front, rssiStrong, 1)
//	// tags will have arrived now, but will still be in the location of the pos sensor
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(posSensor, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure ALL arrivals WERE generated
//	if err := ds.verifyEventPattern(ds.size(), ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//
//}

func TestBasicArrival(t *testing.T) {
	front := nextSensor()

	ds := newTestDataset(lc, 10)

	ds.readAll(front, defaultAntenna, rssiWeak, time.Now(), 1)
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
	ds.readAll(back1, defaultAntenna, rssiMin, time.Now(), 1)
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back1, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure arrival events generated
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}

	// move tags to same facility, different sensor
	ds.readAll(back2, defaultAntenna, rssiStrong, time.Now(), 4)
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
	ds.readAll(back3, defaultAntenna, rssiWeak, time.Now(), 1)
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

	back01 := nextSensor()

	for _, antID := range antennaIds {
		t.Run(fmt.Sprintf("Antenna-%d", antID), func(t *testing.T) {
			ds := newTestDataset(lc, 1)

			// start all tags at initialAntenna
			ds.readAll(back01, initialAntenna, rssiMin, time.Now(), 1)
			ds.sniffEvents()
			// ensure arrival events generated
			if err := ds.verifyEventPattern(1, ArrivedType); err != nil {
				t.Error(err)
			}

			epc := ds.epcs[0]
			tag := ds.tp.inventory[epc]
			// move tag to a different antenna port on same sensor
			ds.readTag(epc, back01, antID, rssiStrong, time.Now(), 4)
			ds.sniffEvents()
			if tag.Location != GetAntennaAlias(back01, antID) {
				t.Errorf("tag location was %s, but we expected %s.\n\t%#v",
					tag.Location, GetAntennaAlias(back01, antID), tag)
			}
			// ensure moved events generated
			if err := ds.verifyEventPattern(1, MovedType); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMoveSameFacility(t *testing.T) {
	ds := newTestDataset(lc, 10)

	back1 := nextSensor()
	back2 := nextSensor()

	// start all tags in the back stock
	ds.readAll(back1, defaultAntenna, rssiMin, time.Now(), 1)
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back1, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), ArrivedType); err != nil {
		t.Error(err)
	}

	// move tag to same facility, different sensor
	ds.readAll(back2, defaultAntenna, rssiStrong, time.Now(), 4)
	ds.sniffEvents()
	if err := ds.verifyAll(Present, GetAntennaAlias(back2, defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), MovedType); err != nil {
		t.Error(err)
	}
}

//func TestMoveDifferentFacility(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(lc, 10)
//
//	front := nextSensor()
//	back := nextSensor()
//
//	// start all tags in the front sales floor
//	ds.readAll(front, rssiMin, 1)
//	ds.updateTagRefs()
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(front, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure arrival events
//	if err := ds.verifyEventPattern(ds.size(), ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// move tag to different facility
//	ds.readAll(back, rssiStrong, 4)
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(back, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved facilities departed/arrival sequence
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//}

//func TestBasicExit(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(lc, 9)
//
//	back := nextSensor()
//	frontExit := nextSensor()
//	front := nextSensor()
//
//	// get it in the system
//	ds.readAll(back, rssiMin, 4)
//	ds.updateTagRefs()
//	ds.clearEvents()
//
//	// one tag read by an EXIT will not make the tag go exiting.
//	ds.readAll(frontExit, rssiMin, 1)
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(back, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure no events generated
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// moving to an exit sensor will put tag in exiting
//	// moving to an exit sensor in another facility will generate departure / arrival
//	ds.readAll(frontExit, rssiWeak, 10)
//	if err := ds.verifyAll(Exiting, sensor.GetAntennaAlias(frontExit, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure departed/arrival events generated for new facility
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// clear exiting by moving to another sensor
//	// done in a loop to simulate being read simultaneously, not 20 on one sensor, and 20 on another
//	for i := 0; i < 20; i++ {
//		ds.readAll(frontExit, rssiMin, 1)
//		ds.readAll(front, rssiStrong, 1)
//	}
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(front, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved events generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	ds.readAll(frontExit, rssiMax, 20)
//	if err := ds.verifyAll(Exiting, sensor.GetAntennaAlias(frontExit, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved events generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//}

//func TestExitingArrivalDepartures(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(lc, 5)
//
//	back := nextSensor()
//	frontExit := nextSensor()
//	front := nextSensor()
//
//	ds.readAll(back, rssiMin, 4)
//	ds.clearEvents()
//
//	ds.updateTagRefs()
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(back, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//
//	// one tag read by an EXIT will not make the tag go exiting.
//	ds.readAll(frontExit, rssiWeak, 1)
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(back, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//
//	// go to exiting state in another facility
//	ds.readAll(frontExit, rssiWeak, 10)
//	if err := ds.verifyAll(Exiting, sensor.GetAntennaAlias(frontExit, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved facilities departed/arrival sequence
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// clear exiting by moving to another sensor
//	ds.readAll(frontExit, rssiMin, 20)
//	ds.readAll(front, rssiStrong, 20)
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(front, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure all moved events were generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// go exiting again
//	ds.readAll(frontExit, rssiMax, 20)
//	if err := ds.verifyAll(Exiting, sensor.GetAntennaAlias(frontExit, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure all moved events were generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//}

//func TestTagDepartAndReturnFromExit(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(lc, 4)
//
//	back := nextSensor()
//	frontExit := nextSensor()
//	front1 := nextSensor()
//
//	ds.readAll(back, rssiMin, 1)
//	ds.updateTagRefs()
//	ds.clearEvents()
//
//	// move to new facility and dampen the rssi from the current sensor
//	ds.readAll(front1, rssiWeak, 20)
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(front1, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure no events were generated
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// move to the exit sensor
//	ds.readAll(frontExit, rssiMax, 20)
//	if err := ds.verifyAll(Exiting, sensor.GetAntennaAlias(frontExit, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure all moved events were generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//
//	// exit personalities do not trigger exiting tags when scheduler
//	// is DYNAMIC and not in MOBILITY which is the default scheduler state
//	// so even though the tag moved to the exit, it is not in the exiting table
//	// todo: missing test code from java?
//}

//func TestTagDepartAndReturnPOS(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(lc, 5)
//
//	back := nextSensor()
//	frontPos := nextSensor()
//	front1 := nextSensor()
//	front2 := nextSensor()
//	front3 := nextSensor()
//
//	// start the tags in the back
//	ds.readAll(back, rssiMin, 1)
//	ds.updateTagRefs()
//	ds.clearEvents()
//
//	// read by the front POS. should still be Present in the back stock
//	ds.setLastReadOnAll(ds.readTimeOrig + (int64(PosDepartedThresholdMillis) / 2))
//	ds.readAll(frontPos, rssiWeak, 1)
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(back, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	// read the tag shortly AFTER the pos DEPART threshold
//	ds.setLastReadOnAll(ds.readTimeOrig + int64(PosDepartedThresholdMillis) + 250)
//	ds.readAll(frontPos, rssiWeak, 1)
//	if err := ds.verifyStateAll(DepartedPos); err != nil {
//		t.Error(err)
//	}
//	// ensure all departed events were generated
//	if err := ds.verifyEventPattern(ds.size(), DepartedEvent); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// and it should stay gone for a while (but not long enough to return)
//	ds.setLastReadOnAll(ds.readTimeOrig + int64(PosReturnThresholdMillis/2))
//	ds.readAll(front1, rssiWeak, 20)
//	if err := ds.verifyStateAll(DepartedPos); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	// keep track of when the tags were departed, because that is what the return threshold is based on
//	lastDeparted := ds.tags[0].LastDeparted
//
//	// read it by another sensor shortly BEFORE pos RETURN threshold
//	ds.setLastReadOnAll(lastDeparted + int64(PosReturnThresholdMillis) - 500)
//	ds.readAll(front2, rssiStrong, 20)
//	if err := ds.verifyStateAll(DepartedPos); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	// read a few tags by the POS sensor shortly AFTER pos RETURN threshold but they should NOT return
//	ds.setLastReadOnAll(lastDeparted + int64(PosReturnThresholdMillis) + 300)
//	ds.readTag(0, frontPos, rssiWeak, 20)
//	ds.readTag(1, frontPos, rssiWeak, 20)
//	if err := ds.verifyState(0, DepartedPos); err != nil {
//		t.Error(err)
//	}
//	if err := ds.verifyState(1, DepartedPos); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	// read it by another sensor shortly AFTER pos RETURN threshold
//	ds.setLastReadOnAll(lastDeparted + int64(PosReturnThresholdMillis) + 1500)
//	ds.readAll(front3, rssiWeak, 20)
//	// note that location is still front2 NOT front3 because it was read stronger by front2
//	if err := ds.verifyAll(Present, sensor.GetAntennaAlias(front2, defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// check for arrival/returned events being generated
//	if err := ds.verifyEventPattern(ds.size(), Returned); err != nil {
//		t.Error(err)
//	}
//	ds.clearEvents()
//
//	// keep track of when the tags were departed, because that is what the return threshold is based on
//	lastArrived := ds.tags[0].LastArrived
//
//	// read it by POS sensor again, and it should depart again
//	ds.setLastReadOnAll(lastArrived + int64(PosDepartedThresholdMillis) + 9999)
//	ds.readAll(frontPos, rssiWeak, 20)
//	if err := ds.verifyStateAll(DepartedPos); err != nil {
//		t.Error(err)
//	}
//	// check for departed events being generated
//	if err := ds.verifyEventPattern(ds.size(), DepartedEvent); err != nil {
//		t.Error(err)
//	}
//}
