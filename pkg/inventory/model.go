/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
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

type EventType string

const (
	ArrivalEvent  EventType = "arrival"
	MovedEvent    EventType = "moved"
	DepartedEvent EventType = "departed"
)

type previousTag struct {
	location     string
	lastRead     int64
	lastDeparted int64
	lastArrived  int64
	state        TagState
}
