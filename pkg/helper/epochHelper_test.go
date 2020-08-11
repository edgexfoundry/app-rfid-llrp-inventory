/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package helper

import (
	"fmt"
	"testing"
	"time"
)

func TestUnixMilli(t *testing.T) {

	var target time.Time

	ms := UnixMilli(target)
	if ms != 0 {
		t.Error("Initial time should be empty")
	}

	target = time.Now()
	ms = UnixMilli(target)
	if ms == 0 {
		t.Error("Initial time should NOT be empty")
	}

	target = time.Now()
	time.Sleep(1 * time.Second)
	ms2 := UnixMilliNow()

	ms = UnixMilli(target)
	fmt.Printf("Time delta: %d\n", ms2-ms)
	if ms2-ms < 900 || ms2-ms > 1100 {
		t.Error("Time calculation bad")
	}
}

func TestUnixMilliCalculation(t *testing.T) {
	expectedMs := int64(1502472327865)
	calcMs := UnixMilli(time.Unix(expectedMs/1000, expectedMs%1000*1000000))
	if calcMs != expectedMs {
		t.Error("Time to epoch calculation failed")
	}
}
