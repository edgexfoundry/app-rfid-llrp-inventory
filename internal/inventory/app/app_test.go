package inventoryapp

import (
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func makeTestApp() (*InventoryApp, *MockConfigClient) {
	app := NewInventoryApp()
	app.lc = getTestingLogger()
	cc := NewMockConfigClient()
	app.configClient = cc
	return app, cc
}

func getTestingLogger() logger.LoggingClient {
	if testing.Verbose() {
		return logger.NewClientStdOut("test", false, "DEBUG")
	}

	return logger.NewMockClient()
}

type bootstrapTest struct {
	name            string
	overwrite       bool
	filename        string
	err             bool
	existingAliases map[string]string
	resultMap       map[string][]byte
	spoofErr        error
}

func TestBootstrapAliasConfig_Table(t *testing.T) {
	tests := []bootstrapTest{
		{
			name:            "Add empty Aliases folder key",
			overwrite:       false,
			filename:        "empty_aliases.toml",
			err:             false,
			existingAliases: nil,
			resultMap: map[string][]byte{
				"Aliases/": nil, // signifies an empty folder
			},
		},
		{
			name:      "Missing Aliases in Toml Error",
			overwrite: false,
			filename:  "missing_aliases_error.toml",
			err:       true,
		},
		{
			name:      "Skip reading toml if data exists",
			overwrite: false,
			// normally this file would throw an error, but because there is existing data
			// we should skip reading the toml file
			filename: "missing_aliases_error.toml",
			err:      false,
			existingAliases: map[string]string{
				"SpeedwayR-10-EF-25_1": "existingAlias",
			},
		},
		{
			name:      "Existing Aliases Do Not Overwrite",
			overwrite: false,
			filename:  "four_aliases.toml",
			err:       false,
			existingAliases: map[string]string{
				"SpeedwayR-10-EF-25_1": "existingAlias",
			},
			// do not expect any added data even though there are 4 values in the toml file
			// because there is existing data and overwrite is false
			resultMap: nil,
		},
		{
			name:      "Do Not Overwrite Existing Empty Folder",
			overwrite: false,
			filename:  "four_aliases.toml",
			err:       false,
			// even though there is no data inside the map, because it is not nil, this signifies
			// that the Aliases folder key exists in Consul
			existingAliases: map[string]string{},
			resultMap:       nil,
		},
		{
			// note: because we are mocking, this is not actually overwriting, just
			// testing that what we read from toml goes into ConfigClient
			name:      "Existing Aliases Overwrite",
			overwrite: true,
			filename:  "one_alias.toml",
			err:       false,
			existingAliases: map[string]string{
				"SpeedwayR-10-EF-25_1": "existingAlias",
			},
			resultMap: map[string][]byte{
				"SpeedwayR-10-EF-25_1": []byte("alias1"),
			},
		},
		{
			name:            "Add Multiple Aliases",
			overwrite:       false,
			filename:        "four_aliases.toml",
			err:             false,
			existingAliases: nil,
			resultMap: map[string][]byte{
				"SpeedwayR-10-EF-25_1": []byte("alias1"),
				"SpeedwayR-10-EF-25_2": []byte("alias2"),
				"SpeedwayR-10-EF-25_3": []byte("alias3"),
				"SpeedwayR-10-EF-25_4": []byte("alias4"),
			},
		},
		{
			// note: because we are mocking, this is not actually overwriting, just
			// testing that what we read from toml goes into ConfigClient
			name:      "Overwrite Multiple Aliases",
			overwrite: true,
			filename:  "four_aliases.toml",
			err:       false,
			existingAliases: map[string]string{
				"SpeedwayR-10-EF-25_1": "existingAlias",
			},
			resultMap: map[string][]byte{
				"SpeedwayR-10-EF-25_1": []byte("alias1"),
				"SpeedwayR-10-EF-25_2": []byte("alias2"),
				"SpeedwayR-10-EF-25_3": []byte("alias3"),
				"SpeedwayR-10-EF-25_4": []byte("alias4"),
			},
		},
		{
			name:      "Spoof Error with ConfigClient.GetConfiguration",
			overwrite: false,
			filename:  "four_aliases.toml",
			err:       true,
			spoofErr:  errors.New("spoof error in ConfigClient.GetConfiguration"),
		},
		{
			name:      "Spoof Error with ConfigClient.PutConfigurationToml",
			overwrite: true,
			filename:  "four_aliases.toml",
			err:       true,
			spoofErr:  errors.New("spoof error in ConfigClient.PutConfigurationToml"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app, cc := makeTestApp()
			flags := MockFlags{
				overwriteConfig: test.overwrite,
				configDirectory: "./testdata",
				profile:         "",
				configFileName:  test.filename,
			}
			cc.config.Aliases = test.existingAliases

			if test.spoofErr != nil {
				cc.nextErr = test.spoofErr
			}

			err := app.bootstrapAliasConfig(flags)
			if err != nil {
				app.lc.Debug(fmt.Sprintf("got error: %v", err))
			}

			if test.err {
				assert.Error(t, err, "Was expecting to see an error")
			} else {
				assert.NoError(t, err, "Expected no error to occur")
			}
			if test.resultMap == nil {
				test.resultMap = make(map[string][]byte)
			}
			assert.EqualValues(t, test.resultMap, cc.valueMap)
		})
	}
}
