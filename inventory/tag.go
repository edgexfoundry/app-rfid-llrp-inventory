/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import "strconv"

type Gen2Read struct {
	Epc       string `json:"epc"`
	Tid       string `json:"tid"`
	User      string `json:"user"`
	Reserved  string `json:"reserved"`
	DeviceId  string `json:"device_id"`
	AntennaId int    `json:"antenna_id"`
	Timestamp int64  `json:"timestamp"`
	Rssi      int    `json:"rssi"`
}

// todo: alias support
func (r *Gen2Read) AsLocation() string {
	return r.DeviceId + "_" + strconv.Itoa(r.AntennaId)
}

type Tag struct {
	Epc      string
	Tid      string
	User     string
	Reserved string

	Location string

	LastRead     int64
	LastArrived  int64
	LastDeparted int64

	State TagState

	deviceStatsMap map[string]*TagStats
}

type TagState string

const (
	Unknown      TagState = "Unknown"
	Present      TagState = "Present"
	Exiting      TagState = "Exiting"
	DepartedExit TagState = "DepartedExit"
	DepartedPos  TagState = "DepartedPos"
)

type Waypoint struct {
	DeviceId  string
	Timestamp int64
}

type History struct {
	Waypoints []Waypoint
	MaxSize   int
}

type Previous struct {
	location     string
	lastRead     int64
	lastArrived  int64
	lastDeparted int64
	state        TagState
}

func NewTag(epc string) *Tag {
	return &Tag{
		Location:       unknown,
		State:          Unknown,
		deviceStatsMap: make(map[string]*TagStats),
		Epc:            epc,
	}
}

func (tag *Tag) asPreviousTag() Previous {
	return Previous{
		location:     tag.Location,
		lastRead:     tag.LastRead,
		lastDeparted: tag.LastDeparted,
		lastArrived:  tag.LastArrived,
		state:        tag.State,
	}
}

func (tag *Tag) update(read *Gen2Read, weighter *rssiAdjuster) {
	// todo: double check the implementation on this code
	// todo: it may not be complete

	srcAlias := read.AsLocation()

	// only set Tid if it is present
	if read.Tid != "" {
		tag.Tid = read.Tid
	}

	// update timestamp
	tag.LastRead = read.Timestamp

	curStats, found := tag.deviceStatsMap[srcAlias]
	if !found {
		curStats = NewTagStats()
		tag.deviceStatsMap[srcAlias] = curStats
	}
	curStats.update(read)

	if tag.Location == srcAlias {
		// nothing to do
		return
	}

	locationStats, found := tag.deviceStatsMap[tag.Location]
	if !found {
		// this means the tag has never been read (somehow)
		tag.Location = srcAlias
		tag.addHistory(read.Timestamp)
	} else if curStats.getCount() > 2 {
		weight := 0.0
		if weighter != nil {
			weight = weighter.getWeight(locationStats.LastRead)
		}

		//logrus.Debugf("%f, %f", curStats.getRssiMeanDBM(), locationStats.getRssiMeanDBM())

		if curStats.getRssiMeanDBM() > locationStats.getRssiMeanDBM()+weight {
			tag.Location = srcAlias
			tag.addHistory(read.Timestamp)
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
		break
	case DepartedExit:
	case DepartedPos:
		tag.LastDeparted = timestamp
		break
	}

	tag.State = newState
}

func (tag *Tag) addHistory(timestamp int64) {
	// todo: implement
}
