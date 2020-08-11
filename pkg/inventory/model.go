/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

type TagState string

const (
	Unknown      TagState = "Unknown"
	Present      TagState = "Present"
	Exiting      TagState = "Exiting"
	DepartedExit TagState = "DepartedExit"
	DepartedPos  TagState = "DepartedPos"
)

type TagDirection string

const (
	Stationary TagDirection = "Stationary"
	Toward     TagDirection = "Toward"
	Away       TagDirection = "Away"
)

type EventType string

const (
	NoEvent    EventType = "none"
	Arrival    EventType = "arrival"
	Moved      EventType = "moved"
	Departed   EventType = "departed"
	Returned   EventType = "returned"
	CycleCount EventType = "cycle_count"
)

type Waypoint struct {
	DeviceID  string
	Timestamp int64
}

type TagHistory struct {
	Waypoints []Waypoint
	MaxSize   int
}

type previousTag struct {
	location     string
	facilityID   string
	lastRead     int64
	lastDeparted int64
	lastArrived  int64
	state        TagState
	direction    TagDirection
}
