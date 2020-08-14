/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/helper"
	"sync"
	"time"
)

type TagProcessor struct {
	lc          logger.LoggingClient
	inventory   map[string]*Tag
	profile     *MobilityProfile
	inventoryMu sync.Mutex
}

func NewTagProcessor(lc logger.LoggingClient) *TagProcessor {
	profile := loadMobilityProfile(lc)
	return &TagProcessor{
		lc:        lc,
		inventory: make(map[string]*Tag),
		profile:   &profile,
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

// ProcessReport
// todo: desc
func (tp *TagProcessor) ProcessReport(r *AccessReport, eventCh chan<- Event) {
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
		tp.process(r.OriginMillis, rt, eventCh)
	}
}

func (tp *TagProcessor) process(referenceTimestamp int64, report *TagReport, eventCh chan<- Event) {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	tag, exists := tp.inventory[report.EPC]
	if !exists {
		tag = NewTag(report.EPC)
		tp.inventory[report.EPC] = tag
	}

	prev := tag.asPreviousTag()
	tag.update(referenceTimestamp, report, tp)

	switch prev.state {

	case Unknown, Departed:
		tag.setState(Present)
		eventCh <- ArrivedEvent{
			EPC:       tag.EPC,
			Timestamp: tag.LastRead,
			Location:  tag.Location,
		}

	case Present:
		if prev.location != "" && prev.location != tag.Location {
			eventCh <- MovedEvent{
				EPC:          tag.EPC,
				Timestamp:    tag.LastRead,
				PrevLocation: prev.location,
				Location:     tag.Location,
			}
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
			// todo: does this need to check departed state?
			numRemoved++
			delete(tp.inventory, epc)
		}
	}

	logrus.Infof("inventory ageout removed %d tags", numRemoved)
	return numRemoved
}

func (tp *TagProcessor) DoAggregateDepartedTask(eventCh chan<- Event) {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	// acquire LOCK BEFORE getting the timestamps, otherwise they can be invalid if we have to wait for the lock
	now := helper.UnixMilliNow()
	expiration := now - int64(AggregateDepartedThresholdMillis)

	for _, tag := range tp.inventory {
		if tag.state == Present && tag.LastRead < expiration {
			tag.setStateAt(Departed, now)
			e := DepartedEvent{
				EPC:          tag.EPC,
				Timestamp:    now,
				LastRead:     tag.LastRead,
				LastLocation: tag.Location,
			}
			// reset the read stats so if it arrives again it will start with fresh data
			tag.resetStats()
			logrus.Debugf("Departed %+v", e)
			eventCh <- e
		}
	}
}
