/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

// TagStats helps keep track of tag read rssi values over time
type TagStats struct {
	LastRead     int64
	readInterval *CircularBuffer
	rssiDbm      *CircularBuffer
}

// NewTagStats returns a new TagStats pointer with circular buffers initialized to the configured default window size
func NewTagStats() *TagStats {
	return &TagStats{
		readInterval: NewCircularBuffer(TagStatsWindowSize),
		rssiDbm:      NewCircularBuffer(TagStatsWindowSize),
	}
}

func (stats *TagStats) update(rssi *float64, lastRead *int64) {
	if rssi != nil {
		stats.rssiDbm.AddValue(*rssi)
	}

	// skip times that are either unknown or at or before the current last read timestamp
	if lastRead == nil || *lastRead <= stats.LastRead {
		return
	}
	if stats.LastRead != 0 {
		stats.readInterval.AddValue(float64(*lastRead - stats.LastRead))
	}
	stats.LastRead = *lastRead
}

func (stats *TagStats) rssiCount() int {
	return stats.rssiDbm.Len()
}
