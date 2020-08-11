/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package sensor

import (
	"testing"
)

func TestSensorAntennaAlias(t *testing.T) {
	tests := []struct {
		deviceID  string
		antennaID int
		expected  string
	}{
		{
			deviceID:  "Sensor-3F7DAC",
			antennaID: 0,
			expected:  "Sensor-3F7DAC_0",
		},
		{
			deviceID:  "Sensor-150000",
			antennaID: 10,
			expected:  "Sensor-150000_10",
		},
		{
			deviceID:  "Sensor-999999",
			antennaID: 3,
			expected:  "Sensor-999999_3",
		},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			s := NewSensor(test.deviceID)
			alias := s.AntennaAlias(test.antennaID)
			if alias != test.expected {
				t.Errorf("Expected alias of %s, but got %s", test.expected, alias)
			}
		})
	}
}
