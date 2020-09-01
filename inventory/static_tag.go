/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

// StaticTag represents a Tag object stuck in time for use with APIs
type StaticTag struct {
	EPC              string                    `json:"epc"`
	TID              string                    `json:"tid"`
	Location         string                    `json:"location"`
	LastRead         int64                     `json:"last_read"`
	LastArrived      int64                     `json:"last_arrived"`
	LastDeparted     int64                     `json:"last_departed"`
	State            TagState                  `json:"state"`
	LocationStatsMap map[string]StaticTagStats `json:"location_stats_map"`
}

// newStaticTag constructs a StaticTag object from an existing Tag pointer
func newStaticTag(tag *Tag) StaticTag {
	s := StaticTag{
		EPC:              tag.EPC,
		TID:              tag.TID,
		Location:         tag.Location,
		LastRead:         tag.LastRead,
		LastArrived:      tag.LastArrived,
		LastDeparted:     tag.LastDeparted,
		State:            tag.state,
		LocationStatsMap: make(map[string]StaticTagStats, len(tag.locationStatsMap)),
	}

	for k, v := range tag.locationStatsMap {
		s.LocationStatsMap[k] = newStaticTagStats(v)
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

// asTagPtr converts a StaticTag back to a Tag pointer for use in restoring inventory.
// It will also restore a basic view of the per-location stats by setting the last read
// timestamp and a single RSSI value which was the previously computed rolling average.
func (s StaticTag) asTagPtr() *Tag {
	t := &Tag{
		EPC:              s.EPC,
		TID:              s.TID,
		Location:         s.Location,
		LastRead:         s.LastRead,
		LastDeparted:     s.LastDeparted,
		LastArrived:      s.LastArrived,
		state:            s.State,
		locationStatsMap: make(map[string]*TagStats),
	}

	// fill in any cached tag stats. this just adds the mean rssi as a single value,
	// so some precision is lost by not having every single value, but it preserves
	// a general view of the data which is good enough for now
	for location, stats := range s.LocationStatsMap {
		tagStats := t.getStats(location)
		tagStats.LastRead = stats.LastRead
		tagStats.rssiDbm.AddValue(stats.MeanRSSI)
	}

	return t
}
