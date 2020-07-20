package routes

import (
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
	contract "github.com/edgexfoundry/go-mod-core-contracts/models"
	"net/http"
	"time"
)

const (
	// SettingsMapKey is used to store the appSetting in the handler's request context
	SettingsKey mapKey = "settingsMap"
	// SettingsHandlerKey is used to store the appSetting in the handler's request context
	SettingsHandlerKey = "settingHandler"
	//StartReadingCommand to start tag reading in readers
	StartReadingCommand = "StartReading"
	//StopReadingCommand to stop tag reading in readers
	StopReadingCommand = "StopReading"
	//ReadCommandKey Read Command Map key
	ReadCommandKey = "readCommand"
	//CoreCommandPUTDevice app settings
	CoreCommandPUTDevice = "CoreCommandPUTDevice"
	//CoreCommandGETDevices app settings
	CoreCommandGETDevices = "CoreCommandGETDevices"
	// LLRPDeviceProfile specifies the name of the device profile in use for LLRP readers, used to determine device type
	LLRPDeviceProfile = "Device.LLRP.Profile"

	ApiConnectionTimeout = 60 * time.Second
)

type mapKey string

// HTTPJSONDeviceListResponse provides list of registered devices/LLRP readers
type HTTPJSONDeviceListResponse struct {
	Content []string `json:"ReaderList"`
}

//SettingsHandler adds a logger and app settings to a response context
type SettingsHandler struct {
	Logger      logger.LoggingClient
	AppSettings map[string]string
}

//GetSettingsHandler will return the logger and app settings
func GetSettingsHandler(req *http.Request) (logger.LoggingClient, map[string]string, error) {
	settingsHandler, ok := req.Context().Value(SettingsKey).(SettingsHandler)
	if !ok {
		return nil, nil, fmt.Errorf("cannot find appsettings")
	}

	loggingClient := settingsHandler.Logger
	appSettings := settingsHandler.AppSettings
	if loggingClient == nil || appSettings == nil {
		return nil, nil, fmt.Errorf("loggingClient/appSettings is nil")
	}

	return loggingClient, appSettings, nil
}

func GetAppSetting(settings map[string]string, name string) (string, error) {
	value, ok := settings[name]

	if ok {
		return value, nil
	}
	return "", errors.Errorf("Application setting %s not found", name)
}

//NewHTTPClient returns HTTP Client variable
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: ApiConnectionTimeout,
	}
}

//GetDeviceList parses response body & sends back list of registered rfid devices
func GetDeviceList(respBody []byte) (deviceList []string, err error) {
	var deviceSlice []contract.Device

	err = json.Unmarshal(respBody, &deviceSlice)
	if err != nil {
		return nil, err
	}

	for _, d := range deviceSlice {

		// filter only llrp readers
		if d.Profile.Name == LLRPDeviceProfile {
			deviceList = append(deviceList, d.Name)
		}
	}
	return deviceList, nil
}

// WriteJSONDeviceListHTTPResponse writes HTTP response in JSON format
func WriteJSONDeviceListHTTPResponse(w http.ResponseWriter, content []string) error {
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(HTTPJSONDeviceListResponse{
		Content: content,
	})
	if err != nil {
		return err
	}

	return nil
}

// WritePlainTextHTTPResponse writes HTTP response in plain text format
func WritePlainTextHTTPResponse(w http.ResponseWriter, content string, statusCode int) error {
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, content)

	return nil
}
