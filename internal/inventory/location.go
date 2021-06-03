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

// NewLocation creates a Location object for the specified device, antenna combo.
func NewLocation(deviceName string, antennaID uint16) Location {
	return Location{DeviceName: deviceName, AntennaID: antennaID}
}

// Equals returns true if the receiver Location has the same device and antenna values as
// the other Location.
func (loc Location) Equals(other Location) bool {
	return loc.AntennaID == other.AntennaID && loc.DeviceName == other.DeviceName
}

// IsEmpty returns true if this Location value has all default values.
func (loc Location) IsEmpty() bool {
	return loc.DeviceName == "" && loc.AntennaID == 0
}

// String returns the string representation of a Location, which is the device name followed by
// an underscore, followed by the antennaID.
func (loc Location) String() string {
	return loc.DeviceName + "_" + strconv.Itoa(int(loc.AntennaID))
}
