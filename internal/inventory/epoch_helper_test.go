//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnixMilli(t *testing.T) {
	var target time.Time

	assert := assert.New(t)

	ms := UnixMilli(target)

	assert.NotEqual(ms, 0, "Initial time should be empty")

	target = time.Now()
	ms = UnixMilli(target)
	assert.NotEqual(ms, 0, "Initial time should NOT be empty")

	target = time.Now()
	time.Sleep(30 * time.Millisecond)
	ms = UnixMilli(target)
	ms2 := UnixMilliNow()

	fmt.Printf("Time delta: %d\n", ms2-ms)

	if ms2-ms < 25 || ms2-ms > 35 {
		t.Error("Time calculation bad")
	}
}

func TestUnixMilliCalculation(t *testing.T) {
	expectedMs := int64(1502472327865)
	calcMs := UnixMilli(time.Unix(expectedMs/1000, expectedMs%1000*1000000))
	assert.Equal(t, calcMs, expectedMs, "Time to epoch calculation failed")
}
