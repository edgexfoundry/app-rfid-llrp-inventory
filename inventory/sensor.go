/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import "strconv"

func makeAlias(deviceID string, antID int) string {
	return deviceID + "_" + strconv.Itoa(antID)
}

// GetAntennaAlias gets the string alias of an Sensor based on the antenna port
// format is DeviceID-AntennaID,  ie. Sensor-150009-0
// If there is an alias defined for that antenna port, use that instead
// Note that each antenna port is supposed to refer to that index in the
// aliases slice
func GetAntennaAlias(deviceID string, antennaID int) string {
	// todo: integrate with alias code
	return makeAlias(deviceID, antennaID)
}
