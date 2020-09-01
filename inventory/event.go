/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

type EventType string

const (
	ArrivedType  EventType = "Arrived"
	MovedType    EventType = "Moved"
	DepartedType EventType = "Departed"
)

type ArrivedEvent struct {
	EPC       string `json:"epc"`
	Timestamp int64  `json:"timestamp"`
	Location  string `json:"location"`
}

type MovedEvent struct {
	EPC          string `json:"epc"`
	Timestamp    int64  `json:"timestamp"`
	PrevLocation string `json:"prev_location"`
	Location     string `json:"location"`
}

type DepartedEvent struct {
	EPC          string `json:"epc"`
	Timestamp    int64  `json:"timestamp"`
	LastRead     int64  `json:"last_read"`
	LastLocation string `json:"last_location"`
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
