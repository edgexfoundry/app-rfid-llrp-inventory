/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import "math"

func rssiToMilliwatts(rssi float64) float64 {
	return math.Pow(10, rssi/10.0)
}

func milliwattsToRssi(mw float64) float64 {
	return math.Log10(mw) * 10.0
}
