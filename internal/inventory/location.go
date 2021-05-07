//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import "strconv"

// Location represents a unique Device-Antenna combination
type Location struct {
	// DeviceName is the name of the device which corresponds to the antenna where the
	// tag is located
	DeviceName string `json:"device_name"`
	// AntennaID is the id number of the antenna as reported by LLRP and is unique/relative to the
	// device it is attached to.
	AntennaID uint16 `json:"antenna_id"`
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
