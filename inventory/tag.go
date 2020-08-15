/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

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
	location string
	lastRead int64
	state    TagState
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
	tag.deviceStatsMap = make(map[string]*TagStats)
}

func (tag *Tag) getStats(location string) *TagStats {
	stats, found := tag.deviceStatsMap[location]
	if !found {
		stats = NewTagStats()
		tag.deviceStatsMap[location] = stats
	}
	return stats
}
