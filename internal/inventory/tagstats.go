//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

const (
	tagStatsWindowSize = 20
)

// tagStats helps keep track of tag read rssi values over time
type tagStats struct {
	lastRead int64
	rssiDbm  *circularBuffer
}

// newTagStats returns a new tagStats pointer with circular buffers initialized to the configured default window size
func newTagStats() *tagStats {
	return &tagStats{
		rssiDbm: newCircularBuffer(tagStatsWindowSize),
	}
}

func (stats *tagStats) updateRSSI(rssi float64) {
	stats.rssiDbm.AddValue(rssi)
}

func (stats *tagStats) updateLastRead(lastRead int64) {
	// skip times that are at or before the current last read timestamp
	if lastRead <= stats.lastRead {
		return
	}
	stats.lastRead = lastRead
}

func (stats *tagStats) rssiCount() int {
	return stats.rssiDbm.Len()
}
