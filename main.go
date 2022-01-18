//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	inventoryapp "edgexfoundry/app-rfid-llrp-inventory/internal/inventory/app"
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
