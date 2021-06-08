//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	inventoryapp "edgexfoundry-holding/rfid-llrp-inventory-service/internal/inventory/app"
	"os"
)

func main() {
	app := inventoryapp.NewInventoryApp()
	if err := app.Initialize(); err != nil {
		app.LoggingClient().Error("App initialization failed", "err", err.Error())
		os.Exit(1)
	}

	if err := app.RunUntilCancelled(); err != nil {
		app.LoggingClient().Error("App RunUntilCancelled failed", "err", err.Error())
		os.Exit(1)
	}
}
