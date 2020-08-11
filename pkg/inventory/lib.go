/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"database/sql"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/helper"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/sensor"
	"sync"
	"time"
)

var (
	inventory   = make(map[string]*Tag)
	exitingTags = make(map[string][]*Tag)

	inventoryMutex = &sync.Mutex{}
)

const (
	unknown         = "UNKNOWN"
	epcEncodeFormat = "tbd"

	PosReturnThresholdMillis         = 0 // todo
	PosDepartedThresholdMillis       = 0 // todo
	AggregateDepartedThresholdMillis = 0 // todo
	AgeOutHours                      = 0 // todo
)

// ProcessInventoryData todo: desc
func ProcessInventoryData(dbs *sql.DB, reading *models.Reading, invData *jsonrpc.InventoryData) (*jsonrpc.InventoryEvent, error) {

	rsp, err := sensor.GetOrCreateRSP(dbs, invData.Params.DeviceID)
	if err != nil {
		return nil, errors.Wrapf(err, "issue trying to retrieve sensor %s from database", invData.Params.DeviceID)
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		logrus.Debugf("sentOn: %v, deviceId: %s, facId: %s, reads: %d, personality: %s, aliases: %v, origin-sent: %v ms, now-origin: %v ms",
			invData.Params.SentOn, rsp.DeviceID, invData.Params.FacilityID, len(invData.Params.Data), rsp.Personality, rsp.Aliases, reading.Origin-invData.Params.SentOn, helper.UnixMilliNow()-reading.Origin)
	}

	facId := invData.Params.FacilityID

	if rsp.FacilityID != facId {
		logrus.Debugf("Updating sensor %s facilityId to %s", rsp.DeviceID, facId)
		rsp.FacilityID = facId
		if err = sensor.Upsert(dbs, rsp); err != nil {
			logrus.Errorf("unable to upsert sensor %s. cause: %v", rsp.DeviceID, err)
		}
	}

	invEvent := jsonrpc.NewInventoryEvent()

	var offset int64
	if config.AppConfig.AdjustLastReadOnByOrigin {
		// offset is an adjustment of timestamps based on when the mqtt-device-service first saw the message compared
		// 		  to when the sensor said it sent it. This can be affected by the latency of the mqtt broker, but hopefully
		//		  that value has relatively low jitter between each packet.
		//		  One thing this will also do is if a sensor thinks it timestamp is in the future, this will
		//		  adjust the times to be standardized against all other sensors in the system.
		offset = reading.Origin - invData.Params.SentOn
		invData.Params.SentOn = reading.Origin
	}

	for _, read := range invData.Params.Data {
		// offset each read (if offset is disabled, this will do nothing)
		read.LastReadOn += offset
		// compare reads based on the time it was received by mqtt-device-service
		processReadData(reading.Origin, invEvent, &read, rsp)
	}

	go func() {
		// do this last to reduce latency in the tag algorithm above
		err = facility.InsertIfNotFound(dbs, facId)
		if err != nil {
			logrus.Errorf("error in finding and inserting facility %s. cause: %v", facId, err)
		}
	}()

	return invEvent, nil
}

func processReadData(referenceTimestamp int64, invEvent *jsonrpc.InventoryEvent, read *jsonrpc.TagRead, rsp *sensor.RSP) {
	inventoryMutex.Lock()
	defer inventoryMutex.Unlock()

	tag, exists := inventory[read.EPC]
	if !exists {
		tag = NewTag(read.EPC)
		inventory[read.EPC] = tag
	}

	prev := tag.asPreviousTag()
	srcAnt := tag.update(referenceTimestamp, rsp, read)

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
		addEvent(invEvent, tag, Arrival)
		break

	case Present:
		if srcAnt.IsPOSAntenna() {
			if !checkDepartPOS(invEvent, tag) {
				checkMovement(invEvent, tag, &prev)
			}
		} else {
			checkExiting(rsp, tag)
			checkMovement(invEvent, tag, &prev)
		}
		break

	case Exiting:
		if srcAnt.IsPOSAntenna() {
			checkDepartPOS(invEvent, tag)
		} else {
			if !srcAnt.IsExitAntenna() && tag.Location == srcAnt.Alias {
				tag.setState(Present)
			}
			checkMovement(invEvent, tag, &prev)
		}
		break

	case DepartedExit:
		if srcAnt.IsPOSAntenna() {
			break
		}

		doTagReturn(invEvent, tag, &prev)
		checkExiting(rsp, tag)
		break

	case DepartedPos:
		if srcAnt.IsPOSAntenna() {
			break
		}

		// Such a tag must remain in the DEPARTED state for
		// a configurable amount of time (i.e. 1 day)
		if tag.LastDeparted < (tag.LastRead - int64(PosReturnThresholdMillis)) {
			doTagReturn(invEvent, tag, &prev)
			checkExiting(rsp, tag)
		}
		break
	}
}

func checkDepartPOS(invEvent *jsonrpc.InventoryEvent, tag *Tag) bool {
	// if tag is ever read by a POS, it immediately generates a departed event
	// as long as it has been seen by our system for a minimum period of time first
	expiration := tag.LastRead - int64(PosDepartedThresholdMillis)

	if tag.LastArrived < expiration {
		tag.setState(DepartedPos)
		addEvent(invEvent, tag, Departed)
		logrus.Debugf("Departed POS: %v", tag)
		return true
	}

	return false
}

func checkMovement(invEvent *jsonrpc.InventoryEvent, tag *Tag, prev *previousTag) {
	if prev.location != "" && prev.location != tag.Location {
		if prev.facilityId != "" && prev.facilityId != tag.FacilityID {
			// change facility (depart old facility, arrive new facility)
			addEventDetails(invEvent, tag.EPC, tag.TID, prev.location, prev.facilityId, Departed, prev.lastRead)
			addEvent(invEvent, tag, Arrival)
		} else {
			addEvent(invEvent, tag, Moved)
		}
	}
}

func checkExiting(ant *sensor.Antenna, tag *Tag) {
	if !ant.IsExitAntenna() || ant.Alias != tag.Location {
		return
	}
	addExiting(ant.FacilityID, tag)
}

func OnSchedulerRunState(runState *jsonrpc.SchedulerRunState) {
	// clear any cached exiting tag status
	logrus.Infof("Scheduler run state has changed to %s. Clearing exiting status of all tags.", runState.Params.RunState)
	clearExiting()
}

func clearExiting() {
	inventoryMutex.Lock()
	defer inventoryMutex.Unlock()

	for _, tags := range exitingTags {
		for _, tag := range tags {
			// test just to be sure, this should not be necessary but belt and suspenders
			if tag.state == Exiting {
				tag.setStateAt(Present, tag.LastArrived)
			}
		}
	}
	exitingTags = make(map[string][]*Tag)
}

func addExiting(facilityId string, tag *Tag) {
	tag.setState(Exiting)

	tags, found := exitingTags[facilityId]
	if !found {
		exitingTags[facilityId] = []*Tag{tag}
	} else {
		exitingTags[facilityId] = append(tags, tag)
	}
}

func doTagReturn(invEvent *jsonrpc.InventoryEvent, tag *Tag, prev *previousTag) {
	if prev.facilityId != "" && prev.facilityId == tag.FacilityID {
		addEvent(invEvent, tag, Returned)
	} else {
		addEvent(invEvent, tag, Arrival)
	}
	tag.setState(Present)
}

func DoAgeoutTask() int {
	inventoryMutex.Lock()
	defer inventoryMutex.Unlock()

	expiration := helper.UnixMilli(time.Now().Add(
		time.Hour * time.Duration(-AgeOutHours)))

	// it is safe to remove from map while iterating in golang
	var numRemoved int
	for epc, tag := range inventory {
		if tag.LastRead < expiration {
			numRemoved++
			delete(inventory, epc)
		}
	}

	logrus.Infof("inventory ageout removed %d tags", numRemoved)
	return numRemoved
}

func DoAggregateDepartedTask() *jsonrpc.InventoryEvent {
	inventoryMutex.Lock()
	defer inventoryMutex.Unlock()

	// acquire lock BEFORE getting the timestamps, otherwise they can be invalid if we have to wait for the lock
	now := helper.UnixMilliNow()
	expiration := now - int64(AggregateDepartedThresholdMillis)

	invEvent := jsonrpc.NewInventoryEvent()

	for _, tags := range exitingTags {
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
				addEvent(invEvent, tag, Departed)
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

func addEvent(invEvent *jsonrpc.InventoryEvent, tag *Tag, event Event) {
	addEventDetails(invEvent, tag.EPC, tag.TID, tag.Location, tag.FacilityID, event, tag.LastRead)
}

func addEventDetails(invEvent *jsonrpc.InventoryEvent, epc string, tid string, location string, facilityId string, event Event, timestamp int64) {
	logrus.Infof("Sending event {epc: %s, tid: %s, event_type: %s, facility_id: %s, location: %s, timestamp: %d}",
		epc, tid, event, facilityId, location, timestamp)

	invEvent.AddTagEvent(jsonrpc.TagEvent{
		Timestamp:       timestamp,
		Location:        location,
		Tid:             tid,
		EpcCode:         epc,
		EpcEncodeFormat: epcEncodeFormat,
		EventType:       string(event),
		FacilityID:      facilityId,
	})
}
