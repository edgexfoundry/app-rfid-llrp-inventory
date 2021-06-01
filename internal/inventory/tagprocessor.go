//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"edgexfoundry-holding/rfid-llrp-inventory-service/internal/llrp"
	"encoding/hex"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"strings"
	"time"
)

type processorConfig struct {
	profile mobilityProfile
	aliases map[string]string

	departedThresholdSeconds uint
	ageOutHours              uint
	adjustLastReadOnByOrigin bool

	// debugLogEnabled is used to be able to only log things when Debug logging is enabled
	// note: this should be something that is able to be determined via the logger.LoggingClient,
	// however currently EdgeX does not support querying the log level
	// see: https://github.com/edgexfoundry/go-mod-core-contracts/issues/294
	debugLogEnabled bool
}

// TagProcessor holds the current inventory data and processes incoming tag read data
type TagProcessor struct {
	lc        logger.LoggingClient
	inventory map[string]*Tag
	config    processorConfig
}

// NewTagProcessor creates a tag processor and pre-loads its mobility profile
func NewTagProcessor(lc logger.LoggingClient, cfg ConsulConfig, tags []StaticTag) *TagProcessor {
	tp := &TagProcessor{
		lc:        lc,
		inventory: make(map[string]*Tag),
	}
	tp.UpdateConfig(cfg)

	for _, t := range tags {
		tp.inventory[t.EPC] = t.asTagPtr()
	}

	return tp
}

// UpdateConfig takes in a ConsulConfig raw config object and converts it into a locally cached
// version that is understood by the TagProcessor. It also generates the correct mobility profile
// based on the supplied values, and the alias map as well.
func (tp *TagProcessor) UpdateConfig(cfg ConsulConfig) {
	as := cfg.ApplicationSettings
	profile := newMobilityProfile(as.MobilityProfileSlope, as.MobilityProfileThreshold, as.MobilityProfileHoldoffMillis)
	aliases := cfg.Aliases
	delete(aliases, "")

	logLevel := strings.ToUpper(cfg.Writable.LogLevel)
	tp.config = processorConfig{
		adjustLastReadOnByOrigin: as.AdjustLastReadOnByOrigin,
		departedThresholdSeconds: as.DepartedThresholdSeconds,
		ageOutHours:              as.AgeOutHours,
		debugLogEnabled:          logLevel == contract.DebugLog || logLevel == contract.TraceLog,
		profile:                  profile,
		aliases:                  aliases,
	}
}

// ProcessReport takes an incoming ROAccessReport and processes each TagReportData.
// For every TagReportData it will update the corresponding tag our in-memory tag database
// based on the latest information.
func (tp *TagProcessor) ProcessReport(r *llrp.ROAccessReport, info ReportInfo) (events []Event, snapshot []StaticTag) {
	if tp.config.adjustLastReadOnByOrigin {
		// offsetMicros is an adjustment of timestamps
		// based on when the device service first saw the message
		// compared to when the sensor said it sent it.
		// This can be affected by the latency,
		// but hopefully that value has relatively little jitter.
		// If a sensor thinks the timestamp is in the future,
		// this will adjust the times to be standardized
		// against all other sensors in the system.
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

	for _, rt := range r.TagReportData {
		if event := tp.processData(&rt, info); event != nil {
			events = append(events, event)
		}
	}
	return events, tp.snapshot()
}

// getAlias returns the alias associated with a location if one has been defined,
// otherwise it returns back the original location.
func (tp *TagProcessor) getAlias(location string) string {
	if alias, exists := tp.config.aliases[location]; exists && alias != "" {
		return alias
	}
	return location
}

// Snapshot takes a snapshot of the entire tag inventory as a slice of StaticTag objects.
// It does this by converting the inventory map of Tag pointers into a flat slice
// of non-pointer StaticTags.
func (tp *TagProcessor) snapshot() []StaticTag {
	res := make([]StaticTag, 0, len(tp.inventory))
	for _, tag := range tp.inventory {
		staticTag := StaticTag{
			EPC:           tag.EPC,
			TID:           tag.TID,
			Location:      tag.Location,
			LocationAlias: tp.getAlias(tag.Location.String()),
			LastRead:      tag.LastRead,
			LastArrived:   tag.LastArrived,
			LastDeparted:  tag.LastDeparted,
			State:         tag.state,
			StatsMap:      make(map[string]StaticTagStats, len(tag.statsMap)),
		}

		// re-populate the stats map
		for loc, stats := range tag.statsMap {
			if stats.rssiCount() == 0 {
				continue // skip empty
			}
			staticTag.StatsMap[loc] = StaticTagStats{
				LastRead: stats.lastRead,
				MeanRSSI: stats.rssiDbm.Mean(),
			}
		}

		res = append(res, staticTag)
	}
	return res
}

// processData processes an incoming TagReportData packet and updates the tag information and
// device stats data structures.
func (tp *TagProcessor) processData(rt *llrp.TagReportData, info ReportInfo) (event Event) {
	var epc string
	if len(rt.EPC96.EPC) > 0 {
		epc = hex.EncodeToString(rt.EPC96.EPC)
	} else {
		epc = hex.EncodeToString(rt.EPCData.EPC)
	}

	tag, exists := tp.inventory[epc]
	if !exists {
		tag = NewTag(epc)
		tp.inventory[epc] = tag
	}
	prevState, prevLoc := tag.state, tag.Location

	// Note: This must be deferred because the code following this defer block has many early-exit
	// scenarios, however we need this deferred block to be run regardless. It is an anonymous
	// function to allow usage of local variables via closure.
	defer func() {
		// Update tag state after processing report.
		switch prevState {
		case Unknown, Departed:
			tag.setState(Present)
			event = ArrivedEvent{
				BaseEvent: BaseEvent{
					EPC:       tag.EPC,
					TID:       tag.TID,
					Timestamp: tag.LastRead,
				},
				Location: tp.getAlias(tag.Location.String()),
			}

		case Present:
			if prevLoc.IsEmpty() || prevLoc.Equals(tag.Location) {
				break
			}

			prevAlias := tp.getAlias(prevLoc.String())
			curAlias := tp.getAlias(tag.Location.String())
			if prevAlias == curAlias {
				break // do not send event if the two locations share the same alias
			}
			event = MovedEvent{
				BaseEvent: BaseEvent{
					EPC:       tag.EPC,
					TID:       tag.TID,
					Timestamp: tag.LastRead,
				},
				OldLocation: prevAlias,
				NewLocation: curAlias,
			}
		}
	}()

	// todo: The following code assumes that if ReadDataAsHex returns ok, that the data contained
	// 		 within is the TID (Tag ID). This is not always the case, but it is the only
	//		 type we currently support for ReadOpSpec.
	if tid, ok := rt.ReadDataAsHex(); ok {
		tag.TID = tid
	}

	hasTimestamp := rt.LastSeenUTC != nil

	var lastRead int64
	if hasTimestamp {
		// offset each read, divide by 1000 to go from microseconds to milliseconds
		lastRead = (int64(*rt.LastSeenUTC) + info.offsetMicros) / 1000

		// only update last read if it is newer
		if lastRead > tag.LastRead {
			tag.LastRead = lastRead
		}
	}

	if rt.AntennaID == nil {
		// if we do not know the antenna id, we cannot compute the location
		return
	}

	readLocation := NewLocation(info.DeviceName, uint16(*rt.AntennaID))
	statsAtReadLoc := tag.getStats(readLocation.String())

	if rssi, hasRSSI := rt.ExtractRSSI(); hasRSSI {
		statsAtReadLoc.updateRSSI(rssi)
	}

	if hasTimestamp {
		statsAtReadLoc.updateLastRead(lastRead)
	}

	if prevLoc.IsEmpty() || tag.Location.Equals(readLocation) {
		tag.Location = readLocation
		return
	}

	statsAtPrevLoc := tag.getStats(tag.Location.String())
	if statsAtPrevLoc.rssiCount() == 0 {
		// Its stats have been cleared; update location.
		tag.Location = readLocation
		return
	}

	// if the incoming read's location has at least 2 data points, lets see if the tag should move
	if statsAtReadLoc.rssiCount() >= 2 {
		if tp.config.debugLogEnabled {
			logReadTiming(tp, info, statsAtPrevLoc, tag)
		}

		locationMean := statsAtPrevLoc.rssiDbm.Mean()
		incomingMean := statsAtReadLoc.rssiDbm.Mean()

		offset := tp.config.profile.computeOffset(info.referenceTimestamp, statsAtPrevLoc.lastRead)
		if tp.config.debugLogEnabled {
			logTagStats(tp, tag, readLocation.String(), incomingMean, locationMean, offset)
		}

		// Update the location if the mean RSSI at the new location
		// is greater than the adjusted mean RSSI of the existing location.
		// Note: This will generate a moved event.
		if incomingMean > (locationMean + offset) {
			tag.Location = readLocation
		}
	}

	return
}

func logTagStats(tp *TagProcessor, tag *Tag, readLocation string, incomingMean float64, existingMean float64, offset float64) {
	tp.lc.Debug("tag stats",
		"epc", tag.EPC,
		"readLoc", readLocation,
		"prevLoc", tag.Location,
		"incomingAvg", fmt.Sprintf("%.2f", incomingMean),
		"existingAvg", fmt.Sprintf("%.2f", existingMean),
		"offset", fmt.Sprintf("%.2f", offset),
		"existingAdjusted", fmt.Sprintf("%.2f", existingMean+offset),
		// if stayFactor is positive, tag will stay, if negative, generates a moved event
		"stayFactor", fmt.Sprintf("%.2f", (existingMean+offset)-incomingMean))
}

func logReadTiming(tp *TagProcessor, info ReportInfo, locationStats *tagStats, tag *Tag) {
	now := UnixMilliNow()
	tp.lc.Debug("read timing",
		"now", now,
		"referenceTimestamp", info.referenceTimestamp,
		"nowMinusRef", fmt.Sprintf("%v", time.Duration(now-info.referenceTimestamp)*time.Millisecond),
		"locationLastRead", locationStats.lastRead,
		"lastRead", tag.LastRead,
		"diff", fmt.Sprintf("%v", time.Duration(tag.LastRead-locationStats.lastRead)*time.Millisecond))
}

// AgeOut is a cleanup method that will remove tag information from our in-memory
// structures if it has not been seen in a long enough time. Only applies to
// tags which are already Departed.
func (tp *TagProcessor) AgeOut() (int, []StaticTag) {
	// subtract the ageOutHours to get the minimum allowed LastRead timestamp.
	// anything older than that is considered aged-out.
	minTimestamp := UnixMilli(time.Now().Add(time.Hour * -time.Duration(tp.config.ageOutHours)))

	// developer note: Go allows us to remove from a map while iterating
	var numRemoved int
	for epc, tag := range tp.inventory {
		if tag.state == Departed && tag.LastRead < minTimestamp {
			numRemoved++
			delete(tp.inventory, epc)
		}
	}

	if numRemoved > 0 {
		tp.lc.Info(fmt.Sprintf("Inventory ageout removed %d tag(s).", numRemoved))
		return numRemoved, tp.snapshot()
	}

	tp.lc.Debug("No tags were aged-out.")
	return 0, nil
}

// AggregateDeparted loops through all tags and sees if any of them should be Departed
// due to not being read in a long enough time.
func (tp *TagProcessor) AggregateDeparted() (events []Event, snapshot []StaticTag) {
	now := time.Now()
	nowMs := now.UnixNano() / 1e6
	// subtract the departedThresholdSeconds to get the minimum allowed LastRead timestamp.
	// anything older than that is considered departed.
	minTimestamp := now.Add(-1*time.Duration(tp.config.departedThresholdSeconds)*time.Second).UnixNano() / 1e6

	for _, tag := range tp.inventory {
		if tag.state == Present && tag.LastRead < minTimestamp {
			tag.setStateAt(Departed, nowMs)
			e := DepartedEvent{
				BaseEvent: BaseEvent{
					EPC:       tag.EPC,
					TID:       tag.TID,
					Timestamp: nowMs,
				},
				LastRead:          tag.LastRead,
				LastKnownLocation: tp.getAlias(tag.Location.String()),
			}

			// reset the read stats so if it arrives again it will start with fresh data
			tag.resetStats()
			tp.lc.Debug("Tag departed.", "epc", tag.EPC, "msSinceLastSeen", nowMs-tag.LastRead)
			events = append(events, e)
		}
	}

	if len(events) == 0 {
		return
	}

	return events, tp.snapshot()
}
