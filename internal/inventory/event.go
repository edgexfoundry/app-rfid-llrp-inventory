//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

type EventType string

const (
	// note: these values are also used when creating the EdgeX event names

	ArrivedType  EventType = "Arrived"
	MovedType    EventType = "Moved"
	DepartedType EventType = "Departed"
)

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

type ArrivedEvent struct {
	BaseEvent
	// Location is the location at which the Arrived event occurred.
	Location string `json:"location"`
}

type MovedEvent struct {
	BaseEvent
	// OldLocation is the previous location at which the tag resided before it moved.
	OldLocation string `json:"old_location"`
	// NewLocation is the current location of the tag after the Moved event.
	NewLocation string `json:"new_location"`
}

type DepartedEvent struct {
	BaseEvent
	// LastRead is the last time in which this tag was read (Unix Epoch milliseconds).
	LastRead int64 `json:"last_read"`
	// LastKnownLocation is the location that the tag was associated with before it Departed.
	LastKnownLocation string `json:"last_known_location"`
}

type Event interface {
	OfType() EventType
}

func (a ArrivedEvent) OfType() EventType {
	return ArrivedType
}

func (m MovedEvent) OfType() EventType {
	return MovedType
}

func (d DepartedEvent) OfType() EventType {
	return DepartedType
}
