//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

// MockFlags implements EdgeX's flags.Common interface for use in unit testing. All values can be
// manually set as needed to setup for the unit test.
type MockFlags struct {
	overwriteConfig bool
	useRegistry     bool
	// TODO: Remove for release v2.0.0 once --registry=<url> no longer supported
	registryUrl       string
	configProviderUrl string
	profile           string
	configDirectory   string
	configFileName    string
}

func (m MockFlags) OverwriteConfig() bool {
	return m.overwriteConfig
}

func (m MockFlags) UseRegistry() bool {
	return m.useRegistry
}

// TODO: Remove for release v2.0.0 once --registry=<url> no longer supported
func (m MockFlags) RegistryUrl() string {
	return m.registryUrl
}

func (m MockFlags) ConfigProviderUrl() string {
	return m.configProviderUrl
}

func (m MockFlags) Profile() string {
	return m.profile
}

func (m MockFlags) ConfigDirectory() string {
	return m.configDirectory
}

func (m MockFlags) ConfigFileName() string {
	return m.configFileName
}

// Not currently needed, so not implemented
func (m MockFlags) Parse([]string) {
	panic("Not implemented.")
}

// Not currently needed, so not implemented
func (m MockFlags) Help() {
	panic("Not implemented.")
}
