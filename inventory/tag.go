/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import "sync"

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

	state            TagState
	locationStatsMap map[string]*TagStats
	statsMu          sync.Mutex
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
