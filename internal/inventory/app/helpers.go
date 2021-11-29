//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"fmt"
	"reflect"

	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/environment"
	"github.com/edgexfoundry/go-mod-bootstrap/v2/bootstrap/flags"
	"github.com/edgexfoundry/go-mod-core-contracts/v2/clients/logger"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
)

const (
	aliasesConfigKey = "Aliases"
)

// loadAliasesFromTomlFile is a helper function that reads just the Aliases config section from
// the user's configuration toml file in order to pre-fill that information into
// the ConfigurationProvider
// Developer Note: This returns nil, nil if Aliases section is found, but no values are present
func loadAliasesFromTomlFile(lc logger.LoggingClient, sdkFlags flags.Common) (*toml.Tree, error) {
	// file path to configuration file is based on the code found in
	// go-mod-bootstrap/config/config.Processor's loadFromFile method
	configDir := environment.GetConfDir(lc, sdkFlags.ConfigDirectory())
	profileDir := environment.GetProfileDir(lc, sdkFlags.Profile())
	configFileName := environment.GetConfigFileName(lc, sdkFlags.ConfigFileName())

	filePath := configDir + "/" + profileDir + configFileName
	lc.Debug(fmt.Sprintf("Loading %s from %s", aliasesConfigKey, filePath))

	tree, err := toml.LoadFile(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "issue loading aliases from toml file")
	}

	aliasesRaw := tree.Get(aliasesConfigKey)
	aliasTree, ok := aliasesRaw.(*toml.Tree)
	if !ok {
		return nil, fmt.Errorf("%s key missing or not a toml tree. type=%v",
			aliasesConfigKey, reflect.TypeOf(aliasesRaw))
	}

	// convert to map[string]interface{} for use in creating nested toml tree below
	aliasMap := aliasTree.ToMap()
	if len(aliasMap) == 0 {
		// if no aliases in the map, return nil
		return nil, nil
	}

	// create a nested structure to mimic the top level config with just the Aliases key
	aliasConfig, err := toml.TreeFromMap(map[string]interface{}{
		aliasesConfigKey: aliasMap,
	})
	if err != nil {
		return nil, errors.Wrap(err, "issue converting aliases tree to nested toml map")
	}

	return aliasConfig, nil
}
