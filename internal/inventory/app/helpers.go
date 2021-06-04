//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"github.com/edgexfoundry/go-mod-bootstrap/bootstrap/flags"
	"github.com/edgexfoundry/go-mod-configuration/configuration"
	"github.com/edgexfoundry/go-mod-configuration/pkg/types"
	"github.com/pkg/errors"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	baseConsulPath = "edgex/appservices/1.0/"
)

// getConfigClient returns a configuration client based on the command line args,
// or a default one if those lack a config provider URL.
// Ideally, a future version of the EdgeX SDKs will give us something like this
// without parsing the args again, but for now, this will do.
func getConfigClient() (configuration.Client, error) {
	sdkFlags := flags.New()
	sdkFlags.Parse(os.Args[1:])
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
