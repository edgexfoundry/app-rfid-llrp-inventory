/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/sensor"
	"os"
	"testing"
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
//	ds := newTestDataset(tp, 10)
//
//	front := generateTestSensor(salesFloor, sensor.NoPersonality)
//	posSensor := generateTestSensor(salesFloor, sensor.POS)
//
//	ds.readAll(posSensor, rssiMin, 1)
//	ds.updateTagRefs()
//	if err := ds.verifyAll(Unknown, posSensor.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	// read a few more times, we still do not want to arrive
//	ds.readAll(posSensor, rssiMin, 4)
//	if err := ds.verifyAll(Unknown, posSensor.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// check no new events
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//
//	ds.readAll(front, rssiStrong, 1)
//	// tags will have arrived now, but will still be in the location of the pos sensor
//	if err := ds.verifyAll(Present, posSensor.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure ALL arrivals WERE generated
//	if err := ds.verifyEventPattern(ds.size(), ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//
//}

func TestBasicArrival(t *testing.T) {
	tp := NewTagProcessor(lc)
	ds := newTestDataset(tp, 10, defaultAntenna)
	front := generateTestSensor(salesFloor, sensor.NoPersonality)

	ds.readAll(front, rssiWeak, 1)
	ds.updateTagRefs()

	if err := ds.verifyAll(Present, front.AntennaAlias(defaultAntenna)); err != nil {
		t.Error(err)
	}

	// ensure ALL arrivals WERE generated
	if err := ds.verifyEventPattern(ds.size(), ArrivalEvent); err != nil {
		t.Error(err)
	}
}

func TestTagMoveWeakRssi(t *testing.T) {
	tp := NewTagProcessor(lc)
	ds := newTestDataset(tp, 10, defaultAntenna)

	back1 := generateTestSensor(backStock, sensor.NoPersonality)
	back2 := generateTestSensor(backStock, sensor.NoPersonality)
	back3 := generateTestSensor(backStock, sensor.NoPersonality)

	// start all tags in the back stock
	ds.readAll(back1, rssiMin, 1)
	ds.updateTagRefs()
	if err := ds.verifyAll(Present, back1.AntennaAlias(defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure arrival events generated
	if err := ds.verifyEventPattern(ds.size(), ArrivalEvent); err != nil {
		t.Error(err)
	}
	ds.resetEvents()

	// move tags to same facility, different sensor
	ds.readAll(back2, rssiStrong, 4)
	if err := ds.verifyAll(Present, back2.AntennaAlias(defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
		t.Error(err)
	}
	ds.resetEvents()

	// test that tag stays at new location even with concurrent reads from weaker sensor
	// MOVE back doesn't happen with weak RSSI
	ds.readAll(back3, rssiWeak, 1)
	if err := ds.verifyAll(Present, back2.AntennaAlias(defaultAntenna)); err != nil {
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

	back01 := generateTestSensor(backStock, sensor.NoPersonality)

	for _, antID := range antennaIds {
		t.Run(fmt.Sprintf("Antenna-%d", antID), func(t *testing.T) {
			tp := NewTagProcessor(lc)
			ds := newTestDataset(tp, 1, initialAntenna)

			// start all tags at antenna port 0
			ds.readAll(back01, rssiMin, 1)
			ds.updateTagRefs()
			// ensure arrival events generated
			if err := ds.verifyEventPattern(1, ArrivalEvent); err != nil {
				t.Error(err)
			}
			ds.resetEvents()

			// move tag to a different antenna port on same sensor
			ds.setAntenna(0, antID)
			ds.readTag(0, back01, rssiStrong, 4)
			if ds.tags[0].Location != back01.AntennaAlias(antID) {
				t.Errorf("tag location was %s, but we expected %s.\n\t%#v",
					ds.tags[0].Location, back01.AntennaAlias(antID), ds.tags[0])
			}
			// ensure moved events generated
			if err := ds.verifyEventPattern(1, MovedEvent); err != nil {
				t.Error(err)
			}
			ds.resetEvents()
		})
	}
}

func TestMoveSameFacility(t *testing.T) {
	tp := NewTagProcessor(lc)
	ds := newTestDataset(tp, 10, defaultAntenna)

	back1 := generateTestSensor(backStock, sensor.NoPersonality)
	back2 := generateTestSensor(backStock, sensor.NoPersonality)

	// start all tags in the back stock
	ds.readAll(back1, rssiMin, 1)
	ds.updateTagRefs()
	if err := ds.verifyAll(Present, back1.AntennaAlias(defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), ArrivalEvent); err != nil {
		t.Error(err)
	}
	ds.resetEvents()

	// move tag to same facility, different sensor
	ds.readAll(back2, rssiStrong, 4)
	if err := ds.verifyAll(Present, back2.AntennaAlias(defaultAntenna)); err != nil {
		t.Error(err)
	}
	// ensure moved events generated
	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
		t.Error(err)
	}
	ds.resetEvents()
}

//func TestMoveDifferentFacility(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(tp, 10)
//
//	front := generateTestSensor(salesFloor, sensor.NoPersonality)
//	back := generateTestSensor(backStock, sensor.NoPersonality)
//
//	// start all tags in the front sales floor
//	ds.readAll(front, rssiMin, 1)
//	ds.updateTagRefs()
//	if err := ds.verifyAll(Present, front.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure arrival events
//	if err := ds.verifyEventPattern(ds.size(), ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//
//	// move tag to different facility
//	ds.readAll(back, rssiStrong, 4)
//	if err := ds.verifyAll(Present, back.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved facilities departed/arrival sequence
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//}

//func TestBasicExit(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(tp, 9)
//
//	back := generateTestSensor(backStock, sensor.NoPersonality)
//	frontExit := generateTestSensor(salesFloor, sensor.Exit)
//	front := generateTestSensor(salesFloor, sensor.NoPersonality)
//
//	// get it in the system
//	ds.readAll(back, rssiMin, 4)
//	ds.updateTagRefs()
//	ds.resetEvents()
//
//	// one tag read by an EXIT will not make the tag go exiting.
//	ds.readAll(frontExit, rssiMin, 1)
//	if err := ds.verifyAll(Present, back.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure no events generated
//	if err := ds.verifyNoEvents(); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//
//	// moving to an exit sensor will put tag in exiting
//	// moving to an exit sensor in another facility will generate departure / arrival
//	ds.readAll(frontExit, rssiWeak, 10)
//	if err := ds.verifyAll(Exiting, frontExit.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure departed/arrival events generated for new facility
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//
//	// clear exiting by moving to another sensor
//	// done in a loop to simulate being read simultaneously, not 20 on one sensor, and 20 on another
//	for i := 0; i < 20; i++ {
//		ds.readAll(frontExit, rssiMin, 1)
//		ds.readAll(front, rssiStrong, 1)
//	}
//	if err := ds.verifyAll(Present, front.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved events generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//
//	ds.readAll(frontExit, rssiMax, 20)
//	if err := ds.verifyAll(Exiting, frontExit.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved events generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//}

//func TestExitingArrivalDepartures(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(tp, 5)
//
//	back := generateTestSensor(backStock, sensor.NoPersonality)
//	frontExit := generateTestSensor(salesFloor, sensor.Exit)
//	front := generateTestSensor(salesFloor, sensor.NoPersonality)
//
//	ds.readAll(back, rssiMin, 4)
//	ds.resetEvents()
//
//	ds.updateTagRefs()
//	if err := ds.verifyAll(Present, back.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//
//	// one tag read by an EXIT will not make the tag go exiting.
//	ds.readAll(frontExit, rssiWeak, 1)
//	if err := ds.verifyAll(Present, back.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//
//	// go to exiting state in another facility
//	ds.readAll(frontExit, rssiWeak, 10)
//	if err := ds.verifyAll(Exiting, frontExit.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure moved facilities departed/arrival sequence
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//
//	// clear exiting by moving to another sensor
//	ds.readAll(frontExit, rssiMin, 20)
//	ds.readAll(front, rssiStrong, 20)
//	if err := ds.verifyAll(Present, front.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure all moved events were generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//
//	// go exiting again
//	ds.readAll(frontExit, rssiMax, 20)
//	if err := ds.verifyAll(Exiting, frontExit.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure all moved events were generated
//	if err := ds.verifyEventPattern(ds.size(), MovedEvent); err != nil {
//		t.Error(err)
//	}
//}

//func TestTagDepartAndReturnFromExit(t *testing.T) {
//	tp := NewTagProcessor(lc)
//	ds := newTestDataset(tp, 4)
//
//	back := generateTestSensor(backStock, sensor.NoPersonality)
//	frontExit := generateTestSensor(salesFloor, sensor.Exit)
//	front1 := generateTestSensor(salesFloor, sensor.NoPersonality)
//
//	ds.readAll(back, rssiMin, 1)
//	ds.updateTagRefs()
//	ds.resetEvents()
//
//	// move to new facility and dampen the rssi from the current sensor
//	ds.readAll(front1, rssiWeak, 20)
//	if err := ds.verifyAll(Present, front1.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// ensure no events were generated
//	if err := ds.verifyEventPattern(2*ds.size(), DepartedEvent, ArrivalEvent); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
//
//	// move to the exit sensor
//	ds.readAll(frontExit, rssiMax, 20)
//	if err := ds.verifyAll(Exiting, frontExit.AntennaAlias(defaultAntenna)); err != nil {
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
//	ds := newTestDataset(tp, 5)
//
//	back := generateTestSensor(backStock, sensor.NoPersonality)
//	frontPos := generateTestSensor(salesFloor, sensor.POS)
//	front1 := generateTestSensor(salesFloor, sensor.NoPersonality)
//	front2 := generateTestSensor(salesFloor, sensor.NoPersonality)
//	front3 := generateTestSensor(salesFloor, sensor.NoPersonality)
//
//	// start the tags in the back
//	ds.readAll(back, rssiMin, 1)
//	ds.updateTagRefs()
//	ds.resetEvents()
//
//	// read by the front POS. should still be Present in the back stock
//	ds.setLastReadOnAll(ds.readTimeOrig + (int64(PosDepartedThresholdMillis) / 2))
//	ds.readAll(frontPos, rssiWeak, 1)
//	if err := ds.verifyAll(Present, back.AntennaAlias(defaultAntenna)); err != nil {
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
//	ds.resetEvents()
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
//	if err := ds.verifyAll(Present, front2.AntennaAlias(defaultAntenna)); err != nil {
//		t.Error(err)
//	}
//	// check for arrival/returned events being generated
//	if err := ds.verifyEventPattern(ds.size(), Returned); err != nil {
//		t.Error(err)
//	}
//	ds.resetEvents()
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
