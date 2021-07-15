//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventoryapp

import (
	"edgexfoundry/app-rfid-llrp-inventory/internal/inventory"
	"fmt"
	"github.com/pelletier/go-toml"
)

// MockConfigClient implements EdgeX's configuration.Client interface for use with unit tests.
// It has the ability to allow the unit test to pre-define the existing configuration that is
// returned, spoof errors to be returned by the API calls, and keeps track of any data
// that has been passed through it for use in validating the data against expected results.
type MockConfigClient struct {
	// config represents the internal config data which is provided via GetConfiguration
	config *inventory.ConsulConfig
	// nextErr holds an error that is to be returned by the next interface method call. this value
	// is cleared every use and should be set before calling an interface method.
	nextErr error

	// tree holds the data provided to PutConfigurationToml method
	tree *toml.Tree
	// valueMap holds any data provided to PutConfigurationValue method
	valueMap map[string][]byte
}

func NewMockConfigClient() *MockConfigClient {
	return &MockConfigClient{
		valueMap: make(map[string][]byte),
		config:   &inventory.ConsulConfig{},
	}
}

// Checks to see if the Configuration service contains the service's configuration.
// Not currently needed, so not implemented
func (m *MockConfigClient) HasConfiguration() (bool, error) {
	panic("Not implemented.")
}

// Puts a full toml configuration into the Configuration service
func (m *MockConfigClient) PutConfigurationToml(configuration *toml.Tree, overwrite bool) error {
	err := m.nextErr
	m.nextErr = nil
	if err != nil {
		return err
	}

	m.tree = configuration
	if configuration.Has(aliasesConfigKey) {
		val, ok := configuration.Get(aliasesConfigKey).(*toml.Tree)
		if !ok {
			panic("unable to convert config to toml.Tree")
		}
		for _, k := range val.Keys() {
			m.valueMap[k] = []byte(fmt.Sprintf("%v", val.Get(k)))
		}
	}
	return nil
}

// Puts a full configuration struct into the Configuration service
// Not currently needed, so not implemented
func (m *MockConfigClient) PutConfiguration(configStruct interface{}, overwrite bool) error {
	panic("Not implemented.")
}

// Gets the full configuration from Consul into the target configuration struct.
// Passed in struct is only a reference for Configuration service. Empty struct is fine
// Returns the configuration in the target struct as interface{}, which caller must cast
func (m *MockConfigClient) GetConfiguration(configStruct interface{}) (interface{}, error) {
	err := m.nextErr
	m.nextErr = nil
	if err != nil {
		return nil, err
	}
	return m.config, nil
}

// Sets up a Consul watch for the target key and send back updates on the update channel.
// Passed in struct is only a reference for Configuration service, empty struct is ok
// Sends the configuration in the target struct as interface{} on updateChannel, which caller must cast
// Not currently needed, so not implemented
func (m *MockConfigClient) WatchForChanges(updateChannel chan<- interface{}, errorChannel chan<- error, configuration interface{}, waitKey string) {
	panic("Not implemented.")
}

// Simply checks if Configuration service is up and running at the configured URL
func (m *MockConfigClient) IsAlive() bool {
	return true
}

// Checks if a configuration value exists in the Configuration service
// Not currently needed, so not implemented
func (m *MockConfigClient) ConfigurationValueExists(name string) (bool, error) {
	panic("Not implemented.")
}

// Gets a specific configuration value from the Configuration service
// Not currently needed, so not implemented
func (m *MockConfigClient) GetConfigurationValue(name string) ([]byte, error) {
	panic("Not implemented.")
}

// Puts a specific configuration value into the Configuration service
func (m *MockConfigClient) PutConfigurationValue(name string, value []byte) error {
	err := m.nextErr
	m.nextErr = nil
	if err != nil {
		return err
	}
	m.valueMap[name] = value
	return nil
}
