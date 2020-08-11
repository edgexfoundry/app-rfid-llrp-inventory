/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package sensor

import (
	"testing"
)

func TestRSPAntennaAlias(t *testing.T) {
	tests := []struct {
		deviceId  string
		antennaId int
		expected  string
	}{
		{
			deviceId:  "Sensor-3F7DAC",
			antennaId: 0,
			expected:  "Sensor-3F7DAC_0",
		},
		{
			deviceId:  "Sensor-150000",
			antennaId: 10,
			expected:  "Sensor-150000_10",
		},
		{
			deviceId:  "Sensor-999999",
			antennaId: 3,
			expected:  "Sensor-999999_3",
		},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			rsp := NewSensor(test.deviceId)
			alias := rsp.AntennaAlias(test.antennaId)
			if alias != test.expected {
				t.Errorf("Expected alias of %s, but got %s", test.expected, alias)
			}
		})
	}
}
