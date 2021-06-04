//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	inventoryapp "edgexfoundry-holding/rfid-llrp-inventory-service/internal/inventory/app"
	"fmt"
	"os"
)

func main() {
	app := inventoryapp.NewInventoryApp()
	if err := app.Initialize(); err != nil {
		fmt.Printf("App initialization failed: %v\n", err)
		os.Exit(1)
	}
	if err := app.RunUntilCancelled(); err != nil {
		fmt.Printf("App RunUntilCancelled failed: %v\n", err)
		os.Exit(1)
	}
}
