/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/helper"
	"time"
)

type TagState string

const (
	Unknown  TagState = "Unknown"
	Present  TagState = "Present"
	Departed TagState = "Departed"
)

type Tag struct {
	EPC string
	TID string

	Location string

	LastRead     int64
	LastDeparted int64
	LastArrived  int64

	state          TagState
	deviceStatsMap map[string]*TagStats
}

type previousTag struct {
	location     string
	lastRead     int64
	state        TagState
}

func NewTag(epc string) *Tag {
	return &Tag{
		EPC:            epc,
		Location:       "",
		state:          Unknown,
		deviceStatsMap: make(map[string]*TagStats),
	}
}

func (tag *Tag) asPreviousTag() previousTag {
	return previousTag{
		location:     tag.Location,
		lastRead:     tag.LastRead,
		state:        tag.state,
	}
}

func (tag *Tag) update(referenceTimestamp int64, report *TagReport, tp *TagProcessor) {
	if report.Antenna == UnknownAntenna {
		return
	}

	srcAlias := GetAntennaAlias(report.DeviceName, report.Antenna)

	// update timestamp
	tag.LastRead = report.LastRead

	// incomingStats represents the statistics for the sensor alias that just read the tag (potential new location)
	incomingStats, found := tag.deviceStatsMap[srcAlias]
	if !found {
		incomingStats = NewTagStats()
		tag.deviceStatsMap[srcAlias] = incomingStats
	}
	incomingStats.update(report, tag.LastRead)

	// todo: apply
	// only set TID if it is present
	//if read.TID != "" {
	//	tag.TID = read.TID
	//}

	if tag.Location == srcAlias {
		// nothing to do
		return
	}

	// locationStats represents the statistics for the tag's current/existing location
	locationStats, found := tag.deviceStatsMap[tag.Location]
	if !found {
		// this means the tag has never been read (somehow)
		tag.Location = srcAlias

	} else if incomingStats.getCount() > 2 {
		now := helper.UnixMilliNow()
		tp.lc.Debug("read timing",
			"now", now,
			"referenceTimestamp", referenceTimestamp,
			"nowMinusRef", fmt.Sprintf("%v", time.Duration(now-referenceTimestamp)*time.Millisecond),
			"locationLastRead", locationStats.LastRead,
			"lastRead", tag.LastRead,
			"diff", fmt.Sprintf("%v", time.Duration(tag.LastRead-locationStats.LastRead)*time.Millisecond))

		weight := tp.profile.ComputeWeight(referenceTimestamp, locationStats.LastRead)
		locationMean := locationStats.rssiDbm.GetMean()
		incomingMean := incomingStats.rssiDbm.GetMean()

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

		// if the new sensor's average is greater than the weighted existing location, generate a moved event
		if incomingMean > (locationMean + weight) {
			tag.Location = srcAlias
		}
	}
}

func (tag *Tag) setState(newState TagState) {
	tag.setStateAt(newState, tag.LastRead)
}

func (tag *Tag) setStateAt(newState TagState, timestamp int64) {
	// capture transition times
	switch newState {
	case Present:
		tag.LastArrived = timestamp
	case Departed:
		tag.LastDeparted = timestamp
	}

	tag.state = newState
}

func (tag *Tag) resetStats() {
	tag.deviceStatsMap = make(map[string]*TagStats)
}

// StaticTag represents a Tag object stuck in time for use with APIs
type StaticTag struct {
	EPC            string                    `json:"epc"`
	TID            string                    `json:"tid"`
	Location       string                    `json:"location"`
	LastRead       int64                     `json:"last_read"`
	LastArrived    int64                     `json:"last_arrived"`
	LastDeparted   int64                     `json:"last_departed"`
	State          TagState                  `json:"state"`
	DeviceStatsMap map[string]StaticTagStats `json:"device_stats_map"`
}

// newStaticTag constructs a StaticTag object from an existing Tag pointer
func newStaticTag(tag *Tag) StaticTag {
	s := StaticTag{
		EPC:            tag.EPC,
		TID:            tag.TID,
		Location:       tag.Location,
		LastRead:       tag.LastRead,
		LastArrived:    tag.LastArrived,
		LastDeparted:   tag.LastDeparted,
		State:          tag.state,
		DeviceStatsMap: make(map[string]StaticTagStats, len(tag.deviceStatsMap)),
	}

	for k, v := range tag.deviceStatsMap {
		s.DeviceStatsMap[k] = newStaticTagStats(v)
	}

	return s
}

// StaticTagStats represents a TagStats object stuck in time for use with APIs
// and includes pre-calculated data
type StaticTagStats struct {
	LastRead int64   `json:"last_read"`
	MeanRSSI float64 `json:"mean_rssi"`
}

// newStaticTagStats constructs a StaticTagStats object from an existing TagStats pointer
func newStaticTagStats(stats *TagStats) StaticTagStats {
	return StaticTagStats{
		LastRead: stats.LastRead,
		MeanRSSI: stats.rssiDbm.GetMean(),
	}
}
