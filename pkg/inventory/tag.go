/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"github.com/sirupsen/logrus"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/jsonrpc"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/sensor"
)

type Tag struct {
	EPC string
	TID string

	Location       string
	DeviceLocation string
	FacilityID     string

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
		DeviceLocation: unknown,
		Direction:      Stationary,
		state:          Unknown,
		deviceStatsMap: make(map[string]*TagStats),
		EPC:            epc,
	}
}

func (tag *Tag) asPreviousTag() previousTag {
	return previousTag{
		location:       tag.Location,
		deviceLocation: tag.DeviceLocation,
		facilityId:     tag.FacilityID,
		lastRead:       tag.LastRead,
		lastDeparted:   tag.LastDeparted,
		lastArrived:    tag.LastArrived,
		state:          tag.state,
		direction:      tag.Direction,
	}
}

func (tag *Tag) update(referenceTimestamp int64, sensor *sensor.Sensor, read *jsonrpc.TagRead) *sensor.Antenna {
	srcAnt := sensor.GetAntenna(read.AntennaID)

	// update timestamp
	tag.LastRead = read.LastReadOn

	// incomingStats represents the statistics for the sensor alias that just read the tag (potential new location)
	incomingStats, found := tag.deviceStatsMap[srcAnt.Alias]
	if !found {
		incomingStats = NewTagStats()
		tag.deviceStatsMap[srcAnt.Alias] = incomingStats
	}
	incomingStats.update(read)

	// only set TID if it is present
	if read.Tid != "" {
		tag.TID = read.Tid
	}

	if tag.Location == srcAnt.Alias {
		// nothing to do
		return
	}

	// locationStats represents the statistics for the tag's current/existing location
	locationStats, found := tag.deviceStatsMap[tag.Location]
	if !found {
		// this means the tag has never been read (somehow)
		tag.Location = srcAnt.Alias
		tag.DeviceLocation = sensor.DeviceID
		tag.FacilityID = sensor.FacilityID

	} else if incomingStats.getCount() > 2 {

		weight := GetMobilityProfile().ComputeWeight(referenceTimestamp, locationStats.LastRead, sensor.IsInDeepScan)
		locationMean := locationStats.rssiDbm.GetMean()
		incomingMean := incomingStats.rssiDbm.GetMean()

		// if the new sensor's average is greater than the weighted existing location, generate a moved event
		if incomingMean > (locationMean + weight) {
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				logrus.Debugf("incoming avg: %f, existing avg: %f, weight: %f, existing adjusted: %f, diff: %f",
					incomingMean, locationMean, weight, locationMean+weight, (locationMean+weight)-incomingMean)
			}

			tag.Location = srcAnt.Alias
			tag.DeviceLocation = sensor.DeviceID
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
