/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

// TagStats helps keep track of tag read rssi values over time
type TagStats struct {
	LastRead int64
	// todo: exported only for ability to marshal to json for now
	ReadInterval *CircularBuffer
	// todo: exported only for ability to marshal to json for now
	RssiMw *CircularBuffer
}

const (
	defaultWindowSize = 20
)

func NewTagStats() *TagStats {
	return &TagStats{
		ReadInterval: NewCircularBuffer(defaultWindowSize),
		RssiMw:       NewCircularBuffer(defaultWindowSize),
	}
}

func (stats *TagStats) update(read *Gen2Read) {
	if stats.LastRead != 0 {
		stats.ReadInterval.AddValue(float64(read.Timestamp - stats.LastRead))
	}
	stats.LastRead = read.Timestamp

	mw := rssiToMilliwatts(float64(read.Rssi) / 10.0)
	stats.RssiMw.AddValue(mw)
}

func (stats *TagStats) getRssiMeanDBM() float64 {
	return milliwattsToRssi(stats.RssiMw.GetMean())
}

func (stats *TagStats) getCount() int {
	return stats.RssiMw.GetCount()
}
