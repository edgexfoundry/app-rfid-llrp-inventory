/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package sensor

import (
	"strconv"
	"sync"
)

const (
	DefaultFacility = "DEFAULT_FACILITY"
)

var (
	sensorMap = map[string]*Sensor{}
	sensorMu  sync.Mutex
)

type Personality string

const (
	NoPersonality Personality = "NONE"
	Exit          Personality = "EXIT"
	POS           Personality = "POS"
	FittingRoom   Personality = "FITTING_ROOM"
)

type Sensor struct {
	DeviceID     string
	FacilityID   string
	UpdatedOn    int64
	IsInDeepScan bool

	antennas map[int]*Antenna
	antMu    sync.Mutex
}

type Antenna struct {
	Personality Personality
	Alias       string
	FacilityID  string
}

func NewSensor(deviceID string) *Sensor {
	sensor := Sensor{
		DeviceID:   deviceID,
		FacilityID: DefaultFacility,
		UpdatedOn:  0,
		antennas:   make(map[int]*Antenna, 0),
	}
	return &sensor
}

func makeAlias(deviceID string, antID int) string {
	return deviceID + "_" + strconv.Itoa(antID)
}

func (s *Sensor) GetAntenna(antID int) *Antenna {
	s.antMu.Lock()
	defer s.antMu.Unlock()

	a, ok := s.antennas[antID]
	if !ok {
		a = &Antenna{
			Personality: NoPersonality,
			Alias:       makeAlias(s.DeviceID, antID),
		}
		s.antennas[antID] = a
	}

	return a
}

// AntennaAlias gets the string alias of an Sensor based on the antenna port
// format is DeviceID-AntennaID,  ie. Sensor-150009-0
// If there is an alias defined for that antenna port, use that instead
// Note that each antenna port is supposed to refer to that index in the
// rsp.Aliases slice
func (s *Sensor) AntennaAlias(antennaID int) string {
	a := s.GetAntenna(antennaID)
	if a.Alias == "" {
		a.Alias = makeAlias(s.DeviceID, antennaID)
	}
	return a.Alias
}

// IsExitAntenna returns true if this Antenna has the EXIT personality
func (a *Antenna) IsExitAntenna() bool {
	return a.Personality == Exit
}

// IsPOSAntenna returns true if this Antenna has the POS personality
func (a *Antenna) IsPOSAntenna() bool {
	return a.Personality == POS
}

func Get(deviceName string) *Sensor {
	sensorMu.Lock()
	defer sensorMu.Unlock()

	s, ok := sensorMap[deviceName]
	if !ok {
		s = NewSensor(deviceName)
		sensorMap[deviceName] = s
	}
	return s
}
