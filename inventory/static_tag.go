/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

// StaticTag represents a Tag object stuck in time for use with APIs
type StaticTag struct {
	EPC            string                    `json:"epc"`
	TID            string                    `json:"tid"`
	Location       string                    `json:"location"`
	LastRead       int64                     `json:"last_read"`
	LastArrived    int64                     `json:"last_arrived"`
	LastDeparted   int64                     `json:"last_departed"`
	State          TagState                  `json:"state"`
	DeviceStatsMap map[string]StaticTagStats `json:"device_stats_map"`
}

// newStaticTag constructs a StaticTag object from an existing Tag pointer
func newStaticTag(tag *Tag) StaticTag {
	s := StaticTag{
		EPC:            tag.EPC,
		TID:            tag.TID,
		Location:       tag.Location,
		LastRead:       tag.LastRead,
		LastArrived:    tag.LastArrived,
		LastDeparted:   tag.LastDeparted,
		State:          tag.state,
		DeviceStatsMap: make(map[string]StaticTagStats, len(tag.deviceStatsMap)),
	}

	for k, v := range tag.deviceStatsMap {
		s.DeviceStatsMap[k] = newStaticTagStats(v)
	}

	return s
}

// StaticTagStats represents a TagStats object stuck in time for use with APIs
// and includes pre-calculated data
type StaticTagStats struct {
	LastRead int64   `json:"last_read"`
	MeanRSSI float64 `json:"mean_rssi"`
}

// newStaticTagStats constructs a StaticTagStats object from an existing TagStats pointer
func newStaticTagStats(stats *TagStats) StaticTagStats {
	return StaticTagStats{
		LastRead: stats.LastRead,
		MeanRSSI: stats.rssiDbm.GetMean(),
	}
}
