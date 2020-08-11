/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/helper"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/sensor"
	"sync"
	"time"
)

const (
	unknown         = "UNKNOWN"
	epcEncodeFormat = "tbd"
)

type TagProcessor struct {
	lc          logger.LoggingClient
	inventory   map[string]*Tag
	exitingTags map[string][]*Tag
	profile     *MobilityProfile
	inventoryMu sync.Mutex
}

func NewTagProcessor(lc logger.LoggingClient) *TagProcessor {
	profile := loadMobilityProfile()
	return &TagProcessor{
		lc:          lc,
		inventory:   make(map[string]*Tag),
		exitingTags: make(map[string][]*Tag),
		profile:     &profile,
	}
}

func (tp *TagProcessor) GetRawInventory() []StaticTag {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	// convert tag map of pointers into a flat array of non-pointers
	res := make([]StaticTag, 0, len(tp.inventory))
	for _, tag := range tp.inventory {
		res = append(res, newStaticTag(tag))
	}
	return res
}

// Process
// todo: desc
func (tp *TagProcessor) Process(report *TagReport) (*jsonrpc.InventoryEvent, error) {
	s := sensor.Get(report.deviceName)
	invEvent := jsonrpc.NewInventoryEvent()
	tp.process(invEvent, report, s)
	return invEvent, nil
}

func (tp *TagProcessor) process(invEvent *jsonrpc.InventoryEvent, report *TagReport, s *sensor.Sensor) {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	epc := report.EPC()
	tag, exists := tp.inventory[epc]
	if !exists {
		tag = NewTag(epc)
		tp.inventory[epc] = tag
	}

	prev := tag.asPreviousTag()
	srcAnt := tag.update(s, report, tp)

	switch prev.state {

	case Unknown:
		// Point of sale NEVER adds new tags to the inventory
		// for the use case of POS reader might be the first
		// sensor in the store hallway to see a tag etc. so
		// need to prevent premature departures
		if srcAnt.IsPOSAntenna() {
			break
		}

		tag.setState(Present)
		tp.addEvent(invEvent, tag, Arrival)
		break

	case Present:
		if srcAnt.IsPOSAntenna() {
			if !tp.checkDepartPOS(invEvent, tag) {
				tp.checkMovement(invEvent, tag, &prev)
			}
		} else {
			tp.checkExiting(srcAnt, tag)
			tp.checkMovement(invEvent, tag, &prev)
		}
		break

	case Exiting:
		if srcAnt.IsPOSAntenna() {
			tp.checkDepartPOS(invEvent, tag)
		} else {
			if !srcAnt.IsExitAntenna() && tag.Location == srcAnt.Alias {
				tag.setState(Present)
			}
			tp.checkMovement(invEvent, tag, &prev)
		}
		break

	case DepartedExit:
		if srcAnt.IsPOSAntenna() {
			break
		}

		tp.doTagReturn(invEvent, tag, &prev)
		tp.checkExiting(srcAnt, tag)
		break

	case DepartedPos:
		if srcAnt.IsPOSAntenna() {
			break
		}

		// Such a tag must remain in the DEPARTED state for
		// a configurable amount of time (i.e. 1 day)
		if tag.LastDeparted < (tag.LastRead - int64(PosReturnThresholdMillis)) {
			tp.doTagReturn(invEvent, tag, &prev)
			tp.checkExiting(srcAnt, tag)
		}
		break
	}
}

func (tp *TagProcessor) checkDepartPOS(invEvent *jsonrpc.InventoryEvent, tag *Tag) bool {
	// if tag is ever read by a POS, it immediately generates a departed event
	// as long as it has been seen by our system for a minimum period of time first
	expiration := tag.LastRead - int64(PosDepartedThresholdMillis)

	if tag.LastArrived < expiration {
		tag.setState(DepartedPos)
		tp.addEvent(invEvent, tag, Departed)
		logrus.Debugf("Departed POS: %v", tag)
		return true
	}

	return false
}

func (tp *TagProcessor) checkMovement(invEvent *jsonrpc.InventoryEvent, tag *Tag, prev *previousTag) {
	if prev.location != "" && prev.location != tag.Location {
		if prev.facilityID != "" && prev.facilityID != tag.FacilityID {
			// change facility (depart old facility, arrive new facility)
			tp.addEventDetails(invEvent, tag.EPC, tag.TID, prev.location, prev.facilityID, Departed, prev.lastRead)
			tp.addEvent(invEvent, tag, Arrival)
		} else {
			tp.addEvent(invEvent, tag, Moved)
		}
	}
}

func (tp *TagProcessor) checkExiting(ant *sensor.Antenna, tag *Tag) {
	if !ant.IsExitAntenna() || ant.Alias != tag.Location {
		return
	}
	tp.addExiting(ant.FacilityID, tag)
}

//func OnSchedulerRunState(runState *jsonrpc.SchedulerRunState) {
//	// clear any cached exiting tag status
//	logrus.Infof("Scheduler run state has changed to %s. Clearing exiting status of all tags.", runState.Params.RunState)
//	clearExiting()
//}

func (tp *TagProcessor) clearExiting() {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	for _, tags := range tp.exitingTags {
		for _, tag := range tags {
			// test just to be sure, this should not be necessary but belt and suspenders
			if tag.state == Exiting {
				tag.setStateAt(Present, tag.LastArrived)
			}
		}
	}
	tp.exitingTags = make(map[string][]*Tag)
}

func (tp *TagProcessor) addExiting(facilityID string, tag *Tag) {
	tag.setState(Exiting)

	tags, found := tp.exitingTags[facilityID]
	if !found {
		tp.exitingTags[facilityID] = []*Tag{tag}
	} else {
		tp.exitingTags[facilityID] = append(tags, tag)
	}
}

func (tp *TagProcessor) doTagReturn(invEvent *jsonrpc.InventoryEvent, tag *Tag, prev *previousTag) {
	if prev.facilityID != "" && prev.facilityID == tag.FacilityID {
		tp.addEvent(invEvent, tag, Returned)
	} else {
		tp.addEvent(invEvent, tag, Arrival)
	}
	tag.setState(Present)
}

func (tp *TagProcessor) DoAgeoutTask() int {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	expiration := helper.UnixMilli(time.Now().Add(
		time.Hour * time.Duration(-AgeOutHours)))

	// it is safe to remove from map while iterating in golang
	var numRemoved int
	for epc, tag := range tp.inventory {
		if tag.LastRead < expiration {
			numRemoved++
			delete(tp.inventory, epc)
		}
	}

	logrus.Infof("inventory ageout removed %d tags", numRemoved)
	return numRemoved
}

func (tp *TagProcessor) DoAggregateDepartedTask() *jsonrpc.InventoryEvent {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	// acquire lock BEFORE getting the timestamps, otherwise they can be invalid if we have to wait for the lock
	now := helper.UnixMilliNow()
	expiration := now - int64(AggregateDepartedThresholdMillis)

	invEvent := jsonrpc.NewInventoryEvent()

	for _, tags := range tp.exitingTags {
		keepIndex := 0
		for _, tag := range tags {

			if tag.state != Exiting {
				// there may be some edge cases where the tag state is invalid
				// skip and do not keep
				continue
			}

			if tag.LastRead < expiration {
				tag.setStateAt(DepartedExit, now)
				logrus.Debugf("Departed %v", tag)
				tp.addEvent(invEvent, tag, Departed)
			} else {
				// if the tag is to be kept, put it back in the slice
				tags[keepIndex] = tag
				keepIndex++
			}
		}
		// shrink to fit actual size
		tags = tags[:keepIndex]
	}

	return invEvent
}

func (tp *TagProcessor) addEvent(invEvent *jsonrpc.InventoryEvent, tag *Tag, eventType EventType) {
	tp.addEventDetails(invEvent, tag.EPC, tag.TID, tag.Location, tag.FacilityID, eventType, tag.LastRead)
}

func (tp *TagProcessor) addEventDetails(invEvent *jsonrpc.InventoryEvent, epc string, tid string, location string, facilityID string, eventType EventType, timestamp int64) {
	tp.lc.Info("Sending event",
		"epc", epc, "tid", tid, "eventType", eventType, "facilityID", facilityID, "location", location, "timestamp", timestamp)

	invEvent.AddTagEvent(jsonrpc.TagEvent{
		Timestamp:       timestamp,
		Location:        location,
		Tid:             tid,
		EpcCode:         epc,
		EpcEncodeFormat: epcEncodeFormat,
		EventType:       string(eventType),
		FacilityID:      facilityID,
	})
}
