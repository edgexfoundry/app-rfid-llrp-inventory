package routes

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
)

type mapKey string

const (
	// SettingsMapKey is used to store the appSetting in the handler's request context
	SettingsMapKey mapKey = "settingsMap"

	// SettingsHandlerKey is used to store the appSetting in the handler's request context
	SettingsHandlerKey = "settingHandler"

	//StartReadingCommand START Reading from devices TODO: To be determined by device profile
	StartReadingCommand = "START_READING"

	//StopReadingCommand STOP Reading from devices TODO: To be determined by device profile
	StopReadingCommand = "STOP_READING"

	//ReadCommand Read Command Map key
	ReadCommand = "readCommand"

	//CoreCommandPUTDevicesNameCommandEndpoint PUT Send Command Endpoint from commands.toml
	CoreCommandPUTDevicesNameCommandEndpoint = "CoreCommandPUTDevicesNameCommandEndpoint"

	//CoreCommandGETDevicesCommandEndpoint GET Send Command Endpoint from commands.toml
	CoreCommandGETDevicesCommandEndpoint = "CoreCommandGETDevicesCommandEndpoint"

	//Limit - Read HTTP Response from EdgeX Core PUT command to N bytes
	Limit = 2000

	//DeviceLimit - Read HTTP Response from EdgeX Core GET Device command to N bytes
	DeviceLimit = 100000
)

// HTTPJSONDeviceListResponse List of Registered Devices
type HTTPJSONDeviceListResponse struct {
	Content []string `json:"SensorList"`
}

//SettingsHandler adds a logger and app settings to a response context
type SettingsHandler struct {
	Logger      logger.LoggingClient
	AppSettings map[string]string
}

//GetSettingsHandler will return the logger and app settings
func GetSettingsHandler(req *http.Request) (logger.LoggingClient, map[string]string, error) {

	settingsMap, ok := req.Context().Value(SettingsMapKey).(map[string]SettingsHandler)
	if !ok || settingsMap == nil {
		return nil, nil, fmt.Errorf("Error: Cannot find appsettings")
	}

	settingsHandlerVar := settingsMap[SettingsHandlerKey]
	logger := settingsHandlerVar.Logger
	appSettings := settingsHandlerVar.AppSettings
	if logger == nil || appSettings == nil {
		return nil, nil, fmt.Errorf("Error: Logger/AppSettings is nil")
	}

	return logger, appSettings, nil

}

//NewHTTPClient returns HTTP Client variable
func NewHTTPClient() *http.Client {

	//DialContext: Setup timeout for an unencrypted HTTP connection
	//TLSHandshakeTimeout: Setup timeout for upgrading the unencrypted connection to an encryped one HTTPS
	//ExpectContinueTimeout: How long you want to wait after you send your payload for the beginning of an answer
	//ResponseHeaderTimeout: How long the complete transfer of the header is allowed to last
	return &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,

			//ExpectContinueTimeout: 4 * time.Second,
			//ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 4 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
		// Prevent endless redirects
		Timeout: 10 * time.Minute,
	}
}
