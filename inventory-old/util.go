/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"math"
	"time"
)

func rssiToMilliwatts(rssi float64) float64 {
	return math.Pow(10, rssi/10.0)
}

func milliwattsToRssi(mw float64) float64 {
	return math.Log10(mw) * 10.0
}

// UnixMilli converts provided time to milliseconds since epoch
func UnixMilli(mytime time.Time) int64 {
	if mytime.Equal(time.Time{}) {
		return 0
	}
	return mytime.UnixNano() / int64(time.Millisecond)
}

// UnixMilli returns current time as milliseconds since epoch
func UnixMilliNow() int64 {
	return UnixMilli(time.Now())
}
