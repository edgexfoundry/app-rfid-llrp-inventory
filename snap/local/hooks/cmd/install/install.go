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
	"path/filepath"

	hooks "github.com/canonical/edgex-snap-hooks/v2"
)

var RES_DIR = "/config/app-rfid-llrp-inventory/res"

func installFile(path string) error {
	destFile := hooks.SnapData + RES_DIR + path
	srcFile := hooks.Snap + RES_DIR + path

	err := os.MkdirAll(filepath.Dir(destFile), 0755)
	if err != nil {
		return err
	}
	err = hooks.CopyFile(srcFile, destFile)

	return err

}

func main() {
	var err error

	if err = hooks.Init(false, "edgex-app-rfid-llrp-inventory"); err != nil {
		fmt.Printf("edgex-app-rfid-llrp-inventory:install: initialization failure: %v\n", err)
		os.Exit(1)

	}

	err = installFile("/configuration.toml")
	if err != nil {
		hooks.Error(fmt.Sprintf("edgex-app-rfid-llrp-inventory:install: %v", err))
		os.Exit(1)
	}
}
