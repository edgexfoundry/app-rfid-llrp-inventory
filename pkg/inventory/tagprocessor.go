/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
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
	unknown = "UNKNOWN"
)

var (
	once   sync.Once
	tagPro *TagProcessor
)

type TagProcessor struct {
	lc          logger.LoggingClient
	inventory   map[string]*Tag
	profile     *MobilityProfile
	inventoryMu sync.Mutex
}

func NewTagProcessor(lc logger.LoggingClient) *TagProcessor {
	once.Do(func() {
		profile := loadMobilityProfile(lc)
		tagPro = &TagProcessor{
			lc:        lc,
			inventory: make(map[string]*Tag),
			profile:   &profile,
		}
	})
	return tagPro
}

func GetRawInventory() []StaticTag {
	tagPro.inventoryMu.Lock()
	defer tagPro.inventoryMu.Unlock()

	// convert tag map of pointers into a flat array of non-pointers
	res := make([]StaticTag, 0, len(tagPro.inventory))
	for _, tag := range tagPro.inventory {
		res = append(res, newStaticTag(tag))
	}
	return res
}

// ProcessReports
// todo: desc
func (tp *TagProcessor) ProcessReports(r *AccessReport) (*jsonrpc.InventoryEvent, error) {
	s := sensor.Get(r.DeviceName)
	invEvent := jsonrpc.NewInventoryEvent()

	var offset int64
	if AdjustLastReadOnByOrigin {
		// offset is an adjustment of timestamps based on when the mqtt-device-service first saw the message compared
		// 		  to when the sensor said it sent it. This can be affected by the latency of the mqtt broker, but hopefully
		//		  that value has relatively low jitter between each packet.
		//		  One thing this will also do is if a sensor thinks it timestamp is in the future, this will
		//		  adjust the times to be standardized against all other sensors in the system.

		var lastRead int64
		for _, rt := range r.TagReports {
			if rt.LastRead > lastRead {
				lastRead = rt.LastRead
			}
		}

		offset = r.OriginMillis - lastRead
	}

	for _, rt := range r.TagReports {
		// offset each read (if offset is disabled, this will do nothing)
		rt.LastRead += offset
		// compare reads based on the time it was received
		tp.process(r.OriginMillis, invEvent, rt, s)
	}
	return invEvent, nil
}

func (tp *TagProcessor) process(referenceTimestamp int64, invEvent *jsonrpc.InventoryEvent, report *TagReport, s *sensor.Sensor) {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	tag, exists := tp.inventory[report.EPC]
	if !exists {
		tag = NewTag(report.EPC)
		tp.inventory[report.EPC] = tag
	}

	prev := tag.asPreviousTag()
	tag.update(referenceTimestamp, s, report, tp)

	switch prev.state {

	case Unknown, Departed:
		tag.setState(Present)
		tp.addEvent(invEvent, tag, ArrivalEvent)

	case Present:
		if prev.location != "" && prev.location != tag.Location {
			tp.addEvent(invEvent, tag, MovedEvent)
		}
	}
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

	// acquire LOCK BEFORE getting the timestamps, otherwise they can be invalid if we have to wait for the lock
	now := helper.UnixMilliNow()
	expiration := now - int64(AggregateDepartedThresholdMillis)

	invEvent := jsonrpc.NewInventoryEvent()
	for _, tag := range tp.inventory {
		if tag.state == Present && tag.LastRead < expiration {
			tag.setStateAt(Departed, now)
			tp.addEvent(invEvent, tag, DepartedEvent)
			logrus.Debugf("Departed %v", tag)
		}
	}

	return invEvent
}

func (tp *TagProcessor) addEvent(invEvent *jsonrpc.InventoryEvent, tag *Tag, eventType EventType) {
	tp.addEventDetails(invEvent, tag.EPC, tag.TID, tag.Location, eventType, tag.LastRead)
}

func (tp *TagProcessor) addEventDetails(invEvent *jsonrpc.InventoryEvent, epc string, tid string, location string, eventType EventType, timestamp int64) {
	tp.lc.Info("Sending event",
		"epc", epc, "tid", tid, "eventType", eventType, "location", location, "timestamp", timestamp)

	invEvent.AddTagEvent(jsonrpc.TagEvent{
		Timestamp: timestamp,
		Location:  location,
		Tid:       tid,
		EpcCode:   epc,
		EventType: string(eventType),
	})
}
