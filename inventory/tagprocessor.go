/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"encoding/hex"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/helper"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"strconv"
	"sync"
	"time"
)

type TagProcessor struct {
	lc      logger.LoggingClient
	eventCh chan<- Event

	inventory   map[string]*Tag
	inventoryMu sync.RWMutex

	cacheMu sync.Mutex

	mobilityProfile *MobilityProfile

	aliases map[string]string
	aliasMu sync.RWMutex
}

func makeDefaultAlias(deviceID string, antID uint16) string {
	return deviceID + "_" + strconv.FormatUint(uint64(antID), 10)
}

// getAlias gets the string alias of a reader based on the antenna port
// Format is DeviceID_AntennaID,  e.g. Reader-EF-10_1
// If there is an alias defined for that antenna port, use that instead
func (tp *TagProcessor) getAlias(deviceID string, antennaID uint16) string {
	defaultAlias := makeDefaultAlias(deviceID, antennaID)

	tp.aliasMu.Lock()
	defer tp.aliasMu.Unlock()

	if alias, exists := tp.aliases[defaultAlias]; exists {
		return alias
	}

	return defaultAlias
}

func (tp *TagProcessor) SetAliases(aliases map[string]string) {
	tp.aliasMu.Lock()
	defer tp.aliasMu.Unlock()

	// aliases configuration map from Consul includes an empty key too for some reason, so is deleted if it exists
	delete(aliases, "")

	tp.aliases = aliases
}

// NewTagProcessor creates a tag processor and pre-loads its mobility profile
func NewTagProcessor(lc logger.LoggingClient, eventCh chan<- Event) *TagProcessor {
	profile := loadMobilityProfile(lc)
	return &TagProcessor{
		lc:              lc,
		eventCh:         eventCh,
		inventory:       make(map[string]*Tag),
		mobilityProfile: &profile,
		aliases:         make(map[string]string),
	}
}

// ReportInfo holds both pre-existing as well as computed metadata about an incoming ROAccessReport
type ReportInfo struct {
	DeviceName  string
	OriginNanos int64

	offsetMicros int64
	// referenceTimestamp is the same as OriginNanos, but converted to milliseconds
	referenceTimestamp int64
}

// NewReportInfo creates a new ReportInfo based on an EdgeX Reading value
func NewReportInfo(reading contract.Reading) ReportInfo {
	return ReportInfo{
		DeviceName:         reading.Device,
		OriginNanos:        reading.Origin,
		referenceTimestamp: reading.Origin / int64(time.Millisecond),
	}
}

// Snapshot takes a snapshot of the entire tag inventory as a slice of StaticTag objects.
// It does this by converting the inventory map of Tag pointers into a flat slice
// of non-pointer StaticTags.
//
// Thread-safe implementation
func (tp *TagProcessor) Snapshot() []StaticTag {
	tp.inventoryMu.RLock()
	defer tp.inventoryMu.RUnlock()

	res := make([]StaticTag, 0, len(tp.inventory))
	for _, tag := range tp.inventory {
		res = append(res, newStaticTag(tag))
	}
	return res
}

// ProcessReport takes an incoming ROAccessReport and processes each TagReportData.
// For every TagReportData it will update the corresponding tag our in-memory tag database
// based on the latest information.
//
// Thread-safe implementation
func (tp *TagProcessor) ProcessReport(r *llrp.ROAccessReport, info ReportInfo) {
	if AdjustLastReadOnByOrigin {
		// offsetMicros is an adjustment of timestamps based on when the mqtt-device-service first saw the message compared
		// 		  to when the sensor said it sent it. This can be affected by the latency of the mqtt broker, but hopefully
		//		  that value has relatively low jitter between each packet.
		//		  One thing this will also do is if a sensor thinks it timestamp is in the future, this will
		//		  adjust the times to be standardized against all other sensors in the system.
		var lastSeenMicros int64
		for _, rt := range r.TagReportData {
			if rt.LastSeenUTC != nil && int64(*rt.LastSeenUTC) > lastSeenMicros {
				lastSeenMicros = int64(*rt.LastSeenUTC)
			}
		}
		if lastSeenMicros > 0 {
			// divide originNanos by 1000 to get to micros
			info.offsetMicros = (info.OriginNanos / 1000) - lastSeenMicros
		}
	}

	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	for _, rt := range r.TagReportData {
		prev, tag := tp.processData(&rt, info)
		// after processing the incoming report, we always need to apply the state machine
		tp.applyStateMachine(prev, tag)
	}
}

// processData processes an incoming TagReportData packet and updates the tag information and
// device stats data structures.
//
// NOTE: Not thread-safe; assumed to only be called while a lock is held on inventoryMu!
func (tp *TagProcessor) processData(rt *llrp.TagReportData, info ReportInfo) (prev previousTag, tag *Tag) {
	var epc string
	if rt.EPC96.EPC != nil && len(rt.EPC96.EPC) > 0 {
		epc = hex.EncodeToString(rt.EPC96.EPC)
	} else {
		epc = hex.EncodeToString(rt.EPCData.EPC)
	}

	tag = tp.getTag(epc)
	prev = tag.asPreviousTag()

	var rssi *float64
	if rt.PeakRSSI != nil {
		peak := float64(*rt.PeakRSSI)
		rssi = &peak
	}

	for _, c := range rt.Custom {
		if VendorIDType(c.VendorID) == Impinj {
			switch ImpinjParamSubtype(c.Subtype) {
			case ImpinjPeakRSSI:
				// todo: peak := float64(c.Data)
				// 		 rssi = &peak
			}
		}
	}

	// todo: parse TID from incoming message
	// only set TID if it is present
	//if read.TID != "" {
	//	tag.TID = read.TID
	//}

	// lastReadPtr will only get set if the last seen timestamp is provided in the report
	var lastReadPtr *int64
	// if LastSeenUTC is not present, we will simply not update the LastRead field
	if rt.LastSeenUTC != nil {
		// offset each read, divide by 1000 to go from microseconds to milliseconds
		lastRead := (int64(*rt.LastSeenUTC) + info.offsetMicros) / 1000
		lastReadPtr = &lastRead

		// only update last read if it is newer
		if lastRead > tag.LastRead {
			tag.LastRead = lastRead
		}
	}

	if rt.AntennaID == nil {
		// if we do not know the antenna id, we cannot compute the location
		return
	}
	srcAlias := tp.getAlias(info.DeviceName, uint16(*rt.AntennaID))

	incomingStats := tag.getStats(srcAlias)
	incomingStats.update(rssi, lastReadPtr)

	if tag.Location == "" {
		// we have not read this tag before, so lets set the initial location; nothing else to do
		tag.Location = srcAlias
		return
	}

	if tag.Location == srcAlias {
		// current location matches incoming read; nothing more to do
		return
	}

	// if the incoming read's location has at least 2 data points, lets see if the tag should move
	if incomingStats.rssiCount() >= 2 {
		// locationStats represents the statistics for the tag's current/existing location
		locationStats := tag.getStats(tag.Location)

		now := helper.UnixMilliNow()
		// todo: only log this when Debug logging is enabled (requires EdgeX to support querying the log level)
		tp.lc.Debug("read timing",
			"now", now,
			"referenceTimestamp", info.referenceTimestamp,
			"nowMinusRef", fmt.Sprintf("%v", time.Duration(now-info.referenceTimestamp)*time.Millisecond),
			"locationLastRead", locationStats.LastRead,
			"lastRead", tag.LastRead,
			"diff", fmt.Sprintf("%v", time.Duration(tag.LastRead-locationStats.LastRead)*time.Millisecond))

		locationMean := locationStats.rssiDbm.GetMean()
		incomingMean := incomingStats.rssiDbm.GetMean()
		weight := tp.mobilityProfile.ComputeWeight(info.referenceTimestamp, locationStats.LastRead)

		// todo: only log this when Debug logging is enabled (requires EdgeX to support querying the log level)
		tp.lc.Debug("tag stats",
			"epc", tag.EPC,
			"incomingLoc", srcAlias,
			"existingLoc", tag.Location,
			"incomingAvg", fmt.Sprintf("%.2f", incomingMean),
			"existingAvg", fmt.Sprintf("%.2f", locationMean),
			"weight", fmt.Sprintf("%.2f", weight),
			"existingAdjusted", fmt.Sprintf("%.2f", locationMean+weight),
			// if stayFactor is positive, tag will stay, if negative, generates a moved event
			"stayFactor", fmt.Sprintf("%.2f", (locationMean+weight)-incomingMean))

		// if the incoming read location's average RSSI is greater than the weighted average
		// RSSI of the existing location, update the location. This will generate a moved event
		// via the state machine
		if incomingMean > (locationMean + weight) {
			tag.Location = srcAlias
		}
	}

	return
}

// getTag will return a pointer to existing tag in the inventory
// or create a new one
//
// NOTE: Not thread-safe; assumed to only be called while a lock is held on inventoryMu!
func (tp *TagProcessor) getTag(epc string) *Tag {
	tag, exists := tp.inventory[epc]
	if !exists {
		tag = NewTag(epc)
		tp.inventory[epc] = tag
	}
	return tag
}

// applyStateMachine will compare the previous tag information with the new information
// and update its state and generate events accordingly.
//
// NOTE: Not thread-safe; assumed to only be called while a lock is held on inventoryMu!
func (tp *TagProcessor) applyStateMachine(prev previousTag, tag *Tag) {
	switch prev.state {

	case Unknown, Departed:
		tag.setState(Present)
		tp.eventCh <- ArrivedEvent{
			EPC:       tag.EPC,
			Timestamp: tag.LastRead,
			Location:  tag.Location,
		}

	case Present:
		if prev.location != "" && prev.location != tag.Location {
			tp.eventCh <- MovedEvent{
				EPC:          tag.EPC,
				Timestamp:    tag.LastRead,
				PrevLocation: prev.location,
				Location:     tag.Location,
			}
		}
	}
}

// RunAgeOutTask is a cleanup method that will remove tag information from our in-memory
// structures if it has not been seen in a long enough time. Only applies to
// tags which are already Departed.
//
// Thread-safe implementation
func (tp *TagProcessor) RunAgeOutTask() int {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	expiration := helper.UnixMilli(time.Now().Add(
		time.Hour * time.Duration(-AgeOutHours)))

	// developer note: Go allows us to remove from a map while iterating
	var numRemoved int
	for epc, tag := range tp.inventory {
		if tag.state == Departed && tag.LastRead < expiration {
			numRemoved++
			delete(tp.inventory, epc)
		}
	}

	tp.lc.Info(fmt.Sprintf("Inventory ageout removed %d tag(s).", numRemoved))
	return numRemoved
}

// RunAggregateDepartedTask loops through all tags and sees if any of them should be Departed
// due to not being read in a long enough time.
//
// Thread-safe implementation
func (tp *TagProcessor) RunAggregateDepartedTask() {
	tp.inventoryMu.Lock()
	defer tp.inventoryMu.Unlock()

	// acquire LOCK BEFORE getting the timestamps, otherwise they can be invalid if we have to wait for the lock
	now := helper.UnixMilliNow()
	expiration := now - int64(DepartedThresholdSeconds*1000)

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
			tp.lc.Debug(fmt.Sprintf("Departed %+v (Last seen %v ago)",
				e, time.Duration(now-tag.LastRead)*time.Millisecond))
			tp.eventCh <- e
		}
	}
}
