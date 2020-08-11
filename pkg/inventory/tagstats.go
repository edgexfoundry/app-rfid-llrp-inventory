/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/pkg/jsonrpc"
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
		readInterval: NewCircularBuffer(config.AppConfig.TagStatsWindowSize),
		rssiDbm:      NewCircularBuffer(config.AppConfig.TagStatsWindowSize),
	}
}

func (stats *TagStats) update(read *jsonrpc.TagRead) {
	if stats.LastRead != 0 {
		stats.readInterval.AddValue(float64(read.LastReadOn - stats.LastRead))
	}
	stats.LastRead = read.LastReadOn

	dbm := float64(read.Rssi) / 10.0
	stats.rssiDbm.AddValue(dbm)
}

func (stats *TagStats) getCount() int {
	return stats.rssiDbm.GetCount()
}
