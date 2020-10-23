//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

const (
	tagStatsWindowSize = 20
)

// TagStats helps keep track of tag read rssi values over time
type TagStats struct {
	LastRead int64
	rssiDbm  *CircularBuffer
}

// NewTagStats returns a new TagStats pointer with circular buffers initialized to the configured default window size
func NewTagStats() *TagStats {
	return &TagStats{
		rssiDbm: NewCircularBuffer(tagStatsWindowSize),
	}
}

func (stats *TagStats) updateRSSI(rssi float64) {
	stats.rssiDbm.AddValue(rssi)
}

func (stats *TagStats) updateLastRead(lastRead int64) {
	// skip times that are at or before the current last read timestamp
	if lastRead <= stats.LastRead {
		return
	}
	stats.LastRead = lastRead
}

func (stats *TagStats) rssiCount() int {
	return stats.rssiDbm.Len()
}
