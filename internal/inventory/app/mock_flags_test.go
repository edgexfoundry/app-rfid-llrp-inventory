package inventoryapp

// MockFlags implements EdgeX's flags.Common interface
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

func (m MockFlags) Parse([]string) {
	panic("Not implemented.")
}

func (m MockFlags) Help() {
	panic("Not implemented.")
}
