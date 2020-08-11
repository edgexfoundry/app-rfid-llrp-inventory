/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package jsonrpc

import (
	"errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/pkg/helper"
)

const (
	inventoryEvent = "inventory_event"
)

type InventoryEvent struct {
	Notification                      // embed
	Params       InventoryEventParams `json:"params"`
}

type InventoryEventParams struct {
	SentOn       int64      `json:"sent_on"`
	ControllerID string     `json:"controller_id,omitempty"` // ok to be empty for handheld
	Data         []TagEvent `json:"data"`
}

// TagEvent is the model of the tag event received from RSP Controller
type TagEvent struct {
	EpcCode         string `json:"epc_code"`
	Tid             string `json:"tid"`
	EpcEncodeFormat string `json:"epc_encode_format"`
	FacilityID      string `json:"facility_id"`
	Location        string `json:"location"`
	EventType       string `json:"event_type,omitempty"`
	Timestamp       int64  `json:"timestamp"`
}

func (invEvent *InventoryEvent) Validate() error {
	if invEvent.IsEmpty() {
		return errors.New("missing data field")
	}

	return invEvent.Notification.Validate()
}

func NewInventoryEvent() *InventoryEvent {
	return &InventoryEvent{
		Notification: Notification{
			Method:  inventoryEvent,
			Version: RpcVersion,
		},
		Params: InventoryEventParams{
			SentOn: helper.UnixMilliNow(),
		},
	}
}

func (invEvent *InventoryEvent) AddTagEvent(event TagEvent) {
	invEvent.Params.Data = append(invEvent.Params.Data, event)
}

func (invEvent *InventoryEvent) IsEmpty() bool {
	return invEvent == nil ||
		invEvent.Params.Data == nil ||
		len(invEvent.Params.Data) == 0
}
