/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/inventory"
	"testing"
)

const (
	ResourceInventoryEventArrived  = "InventoryEventArrived"
	ResourceInventoryEventMoved    = "InventoryEventMoved"
	ResourceInventoryEventDeparted = "InventoryEventDeparted"
)

//goland:noinspection GoBoolExpressions
func TestInventoryEventNames(t *testing.T) {
	if ResourceInventoryEvent+string(inventory.ArrivedType) != ResourceInventoryEventArrived {
		t.Errorf("ResourceInventoryEventArrived is invalid. Was it changed/ranamed?")
	}
	if ResourceInventoryEvent+string(inventory.MovedType) != ResourceInventoryEventMoved {
		t.Errorf("ResourceInventoryEventMoved is invalid. Was it changed/ranamed?")
	}
	if ResourceInventoryEvent+string(inventory.DepartedType) != ResourceInventoryEventDeparted {
		t.Errorf("ResourceInventoryEventDeparted is invalid. Was it changed/ranamed?")
	}
}
