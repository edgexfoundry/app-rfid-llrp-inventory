/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"
	"math"
	"testing"
)

const (
	// floatPrecision is the largest difference allowed for comparing floating point numbers in this test file
	//   we are not using epsilon due to precision loss in the rssi conversions
	floatPrecision = 1e-12
)

func TestRssiConversions(t *testing.T) {
	sampleRssis := []float64{-640, -320, -654, -1000, -290, -126, -1, 0, 1, 100, 333, 950}

	for _, sampleRssi := range sampleRssis {
		t.Run(fmt.Sprintf("Rssi %v", sampleRssi), func(t *testing.T) {
			mw := rssiToMilliwatts(sampleRssi)
			rssi := milliwattsToRssi(mw)

			if math.Abs(sampleRssi-rssi) > floatPrecision {
				t.Errorf("Converting rssi to mw and back resulted in a different value %v dBm -> %v mw -> %v dBm, Diff: %v",
					sampleRssi, mw, rssi, math.Abs(sampleRssi-rssi))
			}
		})
	}
}
