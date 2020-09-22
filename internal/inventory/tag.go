//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
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
	Location     string
	LastRead     int64
	LastDeparted int64
	LastArrived  int64
	state        TagState

	locationStatsMap map[string]*TagStats
	statsMu          sync.Mutex
}

// StaticTag represents a Tag object stuck in time for use with APIs
type StaticTag struct {
	EPC              string                    `json:"epc"`
	TID              string                    `json:"tid"`
	Location         string                    `json:"location"`
	LocationAlias    string                    `json:"location_alias"`
	LastRead         int64                     `json:"last_read"`
	LastArrived      int64                     `json:"last_arrived"`
	LastDeparted     int64                     `json:"last_departed"`
	State            TagState                  `json:"state"`
	LocationStatsMap map[string]StaticTagStats `json:"location_stats_map"`
}

// StaticTagStats represents a TagStats object stuck in time for use with APIs
// and includes pre-calculated data
type StaticTagStats struct {
	LastRead int64   `json:"last_read"`
	MeanRSSI float64 `json:"mean_rssi"`
}

type previousTag struct {
	location string
	lastRead int64
	state    TagState
}

func NewTag(epc string) *Tag {
	return &Tag{
		EPC:              epc,
		Location:         "",
		state:            Unknown,
		locationStatsMap: make(map[string]*TagStats),
	}
}

func (tag *Tag) asPreviousTag() previousTag {
	return previousTag{
		location: tag.Location,
		lastRead: tag.LastRead,
		state:    tag.state,
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

	tag.locationStatsMap = make(map[string]*TagStats)
}

func (tag *Tag) getStats(location string) *TagStats {
	tag.statsMu.Lock()
	defer tag.statsMu.Unlock()

	stats, found := tag.locationStatsMap[location]
	if !found {
		stats = NewTagStats()
		tag.locationStatsMap[location] = stats
	}
	return stats
}

// newStaticTag constructs a StaticTag object from an existing Tag pointer
func (tp *TagProcessor) newStaticTag(tag *Tag) StaticTag {
	s := StaticTag{
		EPC:              tag.EPC,
		TID:              tag.TID,
		Location:         tag.Location,
		LocationAlias:    tp.getAlias(tag.Location),
		LastRead:         tag.LastRead,
		LastArrived:      tag.LastArrived,
		LastDeparted:     tag.LastDeparted,
		State:            tag.state,
		LocationStatsMap: make(map[string]StaticTagStats, len(tag.locationStatsMap)),
	}

	for k, v := range tag.locationStatsMap {
		if v.rssiCount() == 0 {
			// skip empty
			continue
		}
		s.LocationStatsMap[k] = newStaticTagStats(v)
	}

	return s
}

// newStaticTagStats constructs a StaticTagStats object from an existing TagStats pointer
func newStaticTagStats(stats *TagStats) StaticTagStats {
	return StaticTagStats{
		LastRead: stats.LastRead,
		MeanRSSI: stats.rssiDbm.Mean(),
	}
}

// asTagPtr converts a StaticTag back to a Tag pointer for use in restoring inventory.
// It will also restore a basic view of the per-location stats by setting the last read
// timestamp and a single RSSI value which was the previously computed rolling average.
func (s StaticTag) asTagPtr() *Tag {
	t := &Tag{
		EPC:              s.EPC,
		TID:              s.TID,
		Location:         s.Location,
		LastRead:         s.LastRead,
		LastDeparted:     s.LastDeparted,
		LastArrived:      s.LastArrived,
		state:            s.State,
		locationStatsMap: make(map[string]*TagStats),
	}

	// fill in any cached tag stats. this just adds the mean rssi as a single value,
	// so some precision is lost by not having every single value, but it preserves
	// a general view of the data which is good enough for now
	for location, stats := range s.LocationStatsMap {
		tagStats := t.getStats(location)
		tagStats.LastRead = stats.LastRead
		tagStats.rssiDbm.AddValue(stats.MeanRSSI)
	}

	return t
}
