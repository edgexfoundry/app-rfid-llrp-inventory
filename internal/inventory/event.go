//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

// EventType is an enum of the different type of inventory events.
type EventType string

const (
	// note: these values are also used when creating the EdgeX event names

	// ArrivedType defines an inventory event when a tag has Arrived for the first time, or
	// after it has been marked as Departed and is seen again.
	ArrivedType EventType = "Arrived"
	// MovedType defines an inventory event when the tag moves from one Location to another Location.
	MovedType EventType = "Moved"
	// DepartedType defines an inventory event when the tag is not seen for a long period of time.
	DepartedType EventType = "Departed"
)

// BaseEvent is the foundation that all other inventory events are based on and includes the
// values common between all of them.
type BaseEvent struct {
	// EPC stands for Electronic Product Code. EPC was designed as a universal identifier
	// system to provides a unique identity for every physical object in the world.
	EPC string `json:"epc"`
	// TID is commonly referred to as Tag ID or Transponder ID. It is a unique number written to
	// every RFID tag by the manufacturer and is non-writable.
	TID string `json:"tid"`
	// Timestamp is the time at which this event occurred. It represents milliseconds
	// since the Unix Epoch.
	Timestamp int64 `json:"timestamp"`
}

// ArrivedEvent is an inventory event that is generated when a tag is seen for the first time, or
// is seen while in the Departed state.
type ArrivedEvent struct {
	BaseEvent
	// Location is the location at which the Arrived event occurred.
	Location string `json:"location"`
}

// MovedEvent is an inventory event that is generated when a tag moves from one Location to another
// Location.
type MovedEvent struct {
	BaseEvent
	// OldLocation is the previous location at which the tag resided before it moved.
	OldLocation string `json:"old_location"`
	// NewLocation is the current location of the tag after the Moved event.
	NewLocation string `json:"new_location"`
}

// DepartedEvent is an inventory event that is generated when a tag is not seen for a long period
// of time (more than departedThresholdSeconds).
type DepartedEvent struct {
	BaseEvent
	// LastRead is the last time in which this tag was read (Unix Epoch milliseconds).
	LastRead int64 `json:"last_read"`
	// LastKnownLocation is the location that the tag was associated with before it Departed.
	LastKnownLocation string `json:"last_known_location"`
}

// Event is an interface that is implemented to map Event structs to their corresponding
// EventType strings.
type Event interface {
	OfType() EventType
}

// OfType for ArrivedEvent returns ArrivedType
func (a ArrivedEvent) OfType() EventType {
	return ArrivedType
}

// OfType for MovedEvent returns MovedType
func (m MovedEvent) OfType() EventType {
	return MovedType
}

// OfType for DepartedEvent returns DepartedType
func (d DepartedEvent) OfType() EventType {
	return DepartedType
}
