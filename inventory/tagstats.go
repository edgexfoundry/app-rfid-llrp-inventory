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
	rssiMw       *CircularBuffer
}

const (
	defaultWindowSize = 20
)

func NewTagStats() *TagStats {
	return &TagStats{
		readInterval: NewCircularBuffer(defaultWindowSize),
		rssiMw:       NewCircularBuffer(defaultWindowSize),
	}
}

func (stats *TagStats) update(read *Gen2Read) {
	if stats.LastRead != 0 {
		stats.readInterval.AddValue(float64(read.Timestamp - stats.LastRead))
	}
	stats.LastRead = read.Timestamp

	mw := rssiToMilliwatts(float64(read.Rssi) / 10.0)
	stats.rssiMw.AddValue(mw)
}

func (stats *TagStats) getRssiMeanDBM() float64 {
	return milliwattsToRssi(stats.rssiMw.GetMean())
}

func (stats *TagStats) getCount() int {
	return stats.rssiMw.GetCount()
}
