//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"time"
)

// UnixMilli converts provided time to milliseconds since epoch
func UnixMilli(mytime time.Time) int64 {
	if mytime.IsZero() {
		return 0
	}

	return mytime.UnixNano() / 1e6
}

// UnixMilliNow returns current time as milliseconds since epoch
func UnixMilliNow() int64 {
	return time.Now().UnixNano() / 1e6
}
