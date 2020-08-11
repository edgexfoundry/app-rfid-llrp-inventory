/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package helper

import (
	"time"
)

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
