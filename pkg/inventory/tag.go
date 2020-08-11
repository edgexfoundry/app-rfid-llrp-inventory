/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/helper"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/sensor"
)

type Tag struct {
	EPC string
	TID string

	Location   string
	FacilityID string

	LastRead     int64
	LastDeparted int64
	LastArrived  int64

	state     TagState
	Direction TagDirection
	History   []*TagHistory

	deviceStatsMap map[string]*TagStats
}

func NewTag(epc string) *Tag {
	return &Tag{
		Location:       unknown,
		FacilityID:     unknown,
		Direction:      Stationary,
		state:          Unknown,
		deviceStatsMap: make(map[string]*TagStats),
		EPC:            epc,
	}
}

func (tag *Tag) asPreviousTag() previousTag {
	return previousTag{
		location:     tag.Location,
		facilityID:   tag.FacilityID,
		lastRead:     tag.LastRead,
		lastDeparted: tag.LastDeparted,
		lastArrived:  tag.LastArrived,
		state:        tag.state,
		direction:    tag.Direction,
	}
}

func (tag *Tag) update(sensor *sensor.Sensor, report *TagReport, tp *TagProcessor) *sensor.Antenna {
	if report.AntennaID == nil {
		return nil
	}

	srcAnt := sensor.GetAntenna(int(*report.AntennaID))

	// update timestamp
	// todo: utc checking
	if report.LastSeenUTC != nil {
		tag.LastRead = int64(uint64(*report.LastSeenUTC) / uint64(1000))
	}

	// incomingStats represents the statistics for the sensor alias that just read the tag (potential new location)
	incomingStats, found := tag.deviceStatsMap[srcAnt.Alias]
	if !found {
		incomingStats = NewTagStats()
		tag.deviceStatsMap[srcAnt.Alias] = incomingStats
	}
	incomingStats.update(report, tag.LastRead)

	// todo: apply
	// only set TID if it is present
	//if read.TID != "" {
	//	tag.TID = read.TID
	//}

	if tag.Location == srcAnt.Alias {
		// nothing to do
		return srcAnt
	}

	// locationStats represents the statistics for the tag's current/existing location
	locationStats, found := tag.deviceStatsMap[tag.Location]
	if !found {
		// this means the tag has never been read (somehow)
		tag.Location = srcAnt.Alias
		tag.FacilityID = sensor.FacilityID

	} else if incomingStats.getCount() > 2 {

		now := helper.UnixMilliNow()
		weight := tp.profile.ComputeWeight(now, locationStats.LastRead, sensor.IsInDeepScan)
		locationMean := locationStats.rssiDbm.GetMean()
		incomingMean := incomingStats.rssiDbm.GetMean()

		// if the new sensor's average is greater than the weighted existing location, generate a moved event
		if incomingMean > (locationMean + weight) {
			tp.lc.Debug("tag stats",
				"incoming avg", incomingMean,
				"existing avg", locationMean,
				"weight", weight,
				"existing adjusted", locationMean+weight,
				"diff", (locationMean+weight)-incomingMean)

			tag.Location = srcAnt.Alias
			tag.FacilityID = sensor.FacilityID
		}
	}

	return srcAnt
}

func (tag *Tag) setState(newState TagState) {
	tag.setStateAt(newState, tag.LastRead)
}

func (tag *Tag) setStateAt(newState TagState, timestamp int64) {
	// capture transition times
	switch newState {
	case Present:
		tag.LastArrived = timestamp
	case DepartedExit, DepartedPos:
		tag.LastDeparted = timestamp
	}

	tag.state = newState
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
