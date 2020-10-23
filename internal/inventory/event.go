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

type ArrivedEvent struct {
	EPC       string `json:"epc"`
	TID       string `json:"tid"`
	Timestamp int64  `json:"timestamp"`
	Location  string `json:"location"`
}

type MovedEvent struct {
	EPC         string `json:"epc"`
	TID         string `json:"tid"`
	Timestamp   int64  `json:"timestamp"`
	OldLocation string `json:"old_location"`
	NewLocation string `json:"new_location"`
}

type DepartedEvent struct {
	EPC               string `json:"epc"`
	TID               string `json:"tid"`
	Timestamp         int64  `json:"timestamp"`
	LastRead          int64  `json:"last_read"`
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
