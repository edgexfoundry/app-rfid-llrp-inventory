//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"strconv"
	"sync"
)

type TagState string

const (
	Unknown  TagState = "Unknown"
	Present  TagState = "Present"
	Departed TagState = "Departed"
)

type Tag struct {
	EPC          string
	TID          string
	Location     Location
	LastRead     int64
	LastDeparted int64
	LastArrived  int64
	state        TagState

	statsMap map[string]*TagStats
	statsMu  sync.Mutex
}

type Location struct {
	DeviceName string `json:"device_name"`
	AntennaID  uint16 `json:"antenna_id"`
}

func NewLocation(deviceName string, antennaID uint16) Location {
	return Location{DeviceName: deviceName, AntennaID: antennaID}
}

func (loc Location) Equals(other Location) bool {
	return loc.AntennaID == other.AntennaID && loc.DeviceName == other.DeviceName
}

func (loc Location) IsEmpty() bool {
	return loc.DeviceName == "" && loc.AntennaID == 0
}

func (loc Location) String() string {
	return loc.DeviceName + "_" + strconv.Itoa(int(loc.AntennaID))
}

// StaticTag represents a Tag object stuck in time for use with APIs
type StaticTag struct {
	EPC           string                    `json:"epc"`
	TID           string                    `json:"tid"`
	Location      Location                  `json:"location"`
	LocationAlias string                    `json:"location_alias"`
	LastRead      int64                     `json:"last_read"`
	LastArrived   int64                     `json:"last_arrived"`
	LastDeparted  int64                     `json:"last_departed"`
	State         TagState                  `json:"state"`
	StatsMap      map[string]StaticTagStats `json:"stats_map"`
}

// StaticTagStats represents a TagStats object stuck in time for use with APIs
// and includes pre-calculated data
type StaticTagStats struct {
	LastRead int64   `json:"last_read"`
	MeanRSSI float64 `json:"mean_rssi"`
}

func NewTag(epc string) *Tag {
	return &Tag{
		EPC:      epc,
		state:    Unknown,
		statsMap: make(map[string]*TagStats),
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
	tag.statsMu.Lock()
	defer tag.statsMu.Unlock()

	tag.statsMap = make(map[string]*TagStats)
}

func (tag *Tag) getStats(location string) *TagStats {
	tag.statsMu.Lock()
	defer tag.statsMu.Unlock()

	stats, found := tag.statsMap[location]
	if !found {
		stats = NewTagStats()
		tag.statsMap[location] = stats
	}
	return stats
}

// asTagPtr converts a StaticTag back to a Tag pointer for use in restoring inventory.
// It will also restore a basic view of the per-location stats by setting the last read
// timestamp and a single RSSI value which was the previously computed rolling average.
func (s StaticTag) asTagPtr() *Tag {
	t := &Tag{
		EPC:          s.EPC,
		TID:          s.TID,
		Location:     s.Location,
		LastRead:     s.LastRead,
		LastDeparted: s.LastDeparted,
		LastArrived:  s.LastArrived,
		state:        s.State,
		statsMap:     make(map[string]*TagStats),
	}

	// fill in any cached tag stats. this just adds the mean rssi as a single value,
	// so some precision is lost by not having every single value, but it preserves
	// a general view of the data which is good enough for now
	for location, stats := range s.StatsMap {
		tagStats := t.getStats(location)
		tagStats.LastRead = stats.LastRead
		tagStats.rssiDbm.AddValue(stats.MeanRSSI)
	}

	return t
}
