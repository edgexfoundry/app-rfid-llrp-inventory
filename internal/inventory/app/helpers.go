//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-bootstrap/bootstrap/environment"
	"github.com/edgexfoundry/go-mod-bootstrap/bootstrap/flags"
	"github.com/edgexfoundry/go-mod-configuration/configuration"
	"github.com/edgexfoundry/go-mod-configuration/pkg/types"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const (
	aliasesConfigKey = "Aliases"
	baseConsulPath   = "edgex/appservices/1.0/" + serviceKey + "/"
)

// getSdkFlags returns the flags given via command line
func getSdkFlags() *flags.Default {
	sdkFlags := flags.New()
	sdkFlags.Parse(os.Args[1:])
	return sdkFlags
}

// getConfigClient returns a configuration client based on the command line args,
// or a default one if those lack a config provider URL.
// Ideally, a future version of the EdgeX SDKs will give us something like this
// without parsing the args again, but for now, this will do.
func getConfigClient() (configuration.Client, error) {
	sdkFlags := getSdkFlags()
	cpUrl, err := url.Parse(sdkFlags.ConfigProviderUrl())
	if err != nil {
		return nil, err
	}

	cpPort := 8500
	port := cpUrl.Port()
	if port != "" {
		cpPort, err = strconv.Atoi(port)
		if err != nil {
			return nil, errors.Wrap(err, "bad config port")
		}
	}

	configClient, err := configuration.NewConfigurationClient(types.ServiceConfig{
		Host:     cpUrl.Hostname(),
		Port:     cpPort,
		BasePath: baseConsulPath,
		Type:     strings.Split(cpUrl.Scheme, ".")[0],
	})

	return configClient, errors.Wrap(err, "failed to get config client")
}

// loadAliasesFromTomlFile is a helper function that reads just the Aliases config section from
// the user's configuration toml file in order to pre-fill that information into
// the ConfigurationProvider
// Developer Note: This returns nil, nil if Aliases section is found, but no values are present
func loadAliasesFromTomlFile(lc logger.LoggingClient) (*toml.Tree, error) {
	// file path to configuration file is based on the code found in
	// go-mod-bootstrap/config/config.Processor's loadFromFile method
	sdkFlags := getSdkFlags()
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
