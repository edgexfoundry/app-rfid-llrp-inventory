//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"sync"
)

// TagState is an enum of the various states a tag can be in.
type TagState string

const (
	// Unknown is the tag state when then tag has not been read before.
	Unknown TagState = "Unknown"
	// Present is the state a tag will be in once it has been read and not been Departed.
	Present TagState = "Present"
	// Departed is the state a tag is in when it has not been read for a long period time.
	Departed TagState = "Departed"
)

// Tag represents an in-memory view of an RFID tag and its accompanying statistics and metadata.
type Tag struct {
	// EPC stands for Electronic Product Code. EPC was designed as a universal identifier
	// system to provides a unique identity for every physical object in the world.
	EPC string
	// TID is commonly referred to as Tag ID or Transponder ID. It is a unique number written to
	// every RFID tag by the manufacturer and is non-writable.
	TID string
	// Location keeps track of the tag's current location in the form of Device and Antenna combo.
	Location Location
	// LastRead keeps track of the last time the tag was seen by any reader/antenna
	// (Unix Epoch milliseconds). This value is used to determine AgeOut as
	// well as Departed events.
	LastRead int64
	// LastDeparted keeps track of the most recent time this tag was marked as Departed
	// (Unix Epoch milliseconds).
	LastDeparted int64
	// LastArrived keeps track of the most recent time this tag generated an ArrivedEvent.
	// (Unix Epoch milliseconds).
	LastArrived int64

	// state is the current state of the tag (Present, Departed, Unknown)
	state TagState
	// statsMap keeps track of read statistics on a per-antenna basis in order to apply
	// tag location algorithms against.
	statsMap map[string]*tagStats
	// statsMu is a mutex to synchronize access to the statsMap
	statsMu sync.Mutex
}

// NewTag creates a new tag object with the specified EPC. THe state is set to Unknown and
// an empty statsMap is created.
func NewTag(epc string) *Tag {
	return &Tag{
		EPC:      epc,
		state:    Unknown,
		statsMap: make(map[string]*tagStats),
	}
}

func (tag *Tag) setState(newState TagState) {
	tag.setStateAt(newState, tag.LastRead)
}

func (tag *Tag) setStateAt(newState TagState, timestamp int64) {
	// capture transition times
	switch newState {
	case Present:
		tag.LastArrived = timestamp
	case Departed:
		tag.LastDeparted = timestamp
	}

	tag.state = newState
}

func (tag *Tag) resetStats() {
	tag.statsMu.Lock()
	defer tag.statsMu.Unlock()

	tag.statsMap = make(map[string]*tagStats)
}

func (tag *Tag) getStats(location string) *tagStats {
	tag.statsMu.Lock()
	defer tag.statsMu.Unlock()

	stats, found := tag.statsMap[location]
	if !found {
		stats = newTagStats()
		tag.statsMap[location] = stats
	}
	return stats
}
