/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

const (
	TagStatsWindowSize = 20 // todo configure
)

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

func (stats *TagStats) update(report *TagReport, lastRead int64) {
	if stats.LastRead != 0 {
		stats.readInterval.AddValue(float64(lastRead - stats.LastRead))
	}
	stats.LastRead = lastRead

	// todo: what if it is nil?
	if report.PeakRSSI != nil {
		dbm := float64(*report.PeakRSSI)
		stats.rssiDbm.AddValue(dbm)
	}
}

func (stats *TagStats) getCount() int {
	return stats.rssiDbm.GetCount()
}
