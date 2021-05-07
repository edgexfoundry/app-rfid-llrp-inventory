//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

// StaticTag represents a Tag object stuck in time for use with APIs
type StaticTag struct {
	// EPC stands for Electronic Product Code. EPC was designed as a universal identifier
	// system to provides a unique identity for every physical object in the world.
	EPC string `json:"epc"`
	// TID is commonly referred to as Tag ID or Transponder ID. It is a unique number written to
	// every RFID tag by the manufacturer and is non-writable.
	TID string `json:"tid"`
	// Location keeps track of the tag's current location in the form of Device and Antenna combo.
	Location Location `json:"location"`
	// LocationAlias returns the string version of the location adjusted for any user-provided aliases.
	LocationAlias string `json:"location_alias"`
	// LastRead keeps track of the last time the tag was seen by any reader/antenna
	// (Unix Epoch milliseconds). This value is used to determine AgeOut as
	// well as Departed events.
	LastRead int64 `json:"last_read"`
	// LastArrived keeps track of the most recent time this tag generated an ArrivedEvent.
	// (Unix Epoch milliseconds).
	LastArrived int64 `json:"last_arrived"`
	// LastArrived keeps track of the most recent time this tag generated an ArrivedEvent.
	// (Unix Epoch milliseconds).
	LastDeparted int64 `json:"last_departed"`
	// State is the current state of the tag (Present, Departed, Unknown)
	State TagState `json:"state"`
	// StatsMap keeps track of read statistics on a per-antenna basis in order to apply
	// tag location algorithms against.
	StatsMap map[string]StaticTagStats `json:"stats_map"`
}

// StaticTagStats represents a tagStats object stuck in time for use with APIs
// and includes pre-calculated data
type StaticTagStats struct {
	LastRead int64   `json:"last_read"`
	MeanRSSI float64 `json:"mean_rssi"`
}

// asTagPtr converts a StaticTag back to a Tag pointer for use in restoring inventory.
// It will also restore a basic view of the per-location stats by setting the last read
// timestamp and a single RSSI value which was the previously computed rolling average.
func (s StaticTag) asTagPtr() *Tag {
	t := &Tag{
		EPC:          s.EPC,
		TID:          s.TID,
		Location:     s.Location,
		LastRead:     s.LastRead,
		LastDeparted: s.LastDeparted,
		LastArrived:  s.LastArrived,
		state:        s.State,
		statsMap:     make(map[string]*tagStats),
	}

	// fill in any cached tag stats. this just adds the mean rssi as a single value,
	// so some precision is lost by not having every single value, but it preserves
	// a general view of the data which is good enough for now
	for location, stats := range s.StatsMap {
		tagStats := t.getStats(location)
		tagStats.lastRead = stats.LastRead
		tagStats.rssiDbm.AddValue(stats.MeanRSSI)
	}

	return t
}
