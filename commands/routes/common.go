package routes

import (
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

const (
	// SettingsMapKey is used to store the appSetting in the handler's request context
	SettingsKey mapKey = "settingsMap"

	// SettingsHandlerKey is used to store the appSetting in the handler's request context
	SettingsHandlerKey = "settingHandler"

	////StartReadingCommand to start tag reading in readers
	//StartReadingCommand = "START_READING"
	//
	////StopReadingCommand to stop tag reading in readers
	//StopReadingCommand = "STOP_READING"
	//
	////ReadCommand Read Command Map key
	//ReadCommand = "readCommand"

	//CoreCommandPUTDevice app settings
	CoreCommandPUTDevice = "CoreCommandPUTDevice"

	//CoreCommandGETDevices app settings
	CoreCommandGETDevices = "CoreCommandGETDevices"

	ApiConnectionTimeout = 60 * time.Second
)

// Device list from edgex
type Device struct {
	Name string `json:"name"`
}

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

	logger := settingsHandler.Logger
	appSettings := settingsHandler.AppSettings
	if logger == nil || appSettings == nil {
		return nil, nil, fmt.Errorf("logger/appSettings is nil")
	}

	return logger, appSettings, nil
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
	var deviceSlice []Device

	err = json.Unmarshal(respBody, &deviceSlice)
	if err != nil {
		return nil, err
	}

	for _, result := range deviceSlice {
		deviceList = append(deviceList, result.Name)
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
