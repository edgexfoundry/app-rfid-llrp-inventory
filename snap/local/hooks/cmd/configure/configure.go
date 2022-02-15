// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2021 Canonical Ltd
 *
 *  Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
 *  in compliance with the License. You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under the License
 * is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
 * or implied. See the License for the specific language governing permissions and limitations under
 * the License.
 *
 * SPDX-License-Identifier: Apache-2.0'
 */

package main

import (
	"fmt"
	"os"
	"strings"

	hooks "github.com/canonical/edgex-snap-hooks/v2"
)

func main() {
	var debug = false
	var enable = true
	var err error
	var envJSON string
	var cli *hooks.CtlCli = hooks.NewSnapCtl()

	status, err := cli.Config("debug")
	if err != nil {
		fmt.Println(fmt.Sprintf("edgex-asc:configure: can't read value of 'debug': %v", err))
		os.Exit(1)
	}
	if status == "true" {
		debug = true
	}

	if err = hooks.Init(debug, "app-rfid-llrp-inventory"); err != nil {
		fmt.Println(fmt.Sprintf("edgex-app-rfid-llrp-inventory:configure: initialization failure: %v", err))
		os.Exit(1)

	}

	envJSON, err = cli.Config(hooks.EnvConfig)
	if err != nil {
		hooks.Error(fmt.Sprintf("Reading config 'env' failed: %v", err))
		os.Exit(1)
	}

	err = hooks.HandleEdgeXConfig("app-rfid-llrp-inventory", envJSON, nil)
	if err != nil {
		hooks.Error(fmt.Sprintf("HandleEdgeXConfig failed: %v", err))
		os.Exit(1)
	}

	// If autostart is not explicitly set, default to "no"
	autostart, err := cli.Config(hooks.AutostartConfig)
	if err != nil {
		hooks.Error(fmt.Sprintf("Reading config 'autostart' failed: %v", err))
		os.Exit(1)
	}
	if autostart == "" {
		hooks.Debug("edgex-app-rfid-llrp-inventory: autostart is NOT set, initializing to 'no'")
		autostart = "no"
	}

	autostart = strings.ToLower(autostart)
	if autostart == "true" || autostart == "yes" {
		enable = false
	} else if autostart == "false" || autostart == "no" {
		enable = false
	} else {
		hooks.Error(fmt.Sprintf("Invalid value for 'autostart' : %s", autostart))
		os.Exit(1)
	}

	// service is stopped/disabled by default in the install hook
	if enable {
		err = cli.Start("app-rfid-llrp-inventory", true)
		if err != nil {
			hooks.Error(fmt.Sprintf("Can't start service - %v", err))
			os.Exit(1)
		}
	}
}
