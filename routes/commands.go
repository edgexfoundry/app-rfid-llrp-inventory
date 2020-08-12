package routes

import (
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"sync"
)

var (
	client      = NewHTTPClient()
	_indexBytes []byte
)

// indexBytes lazy-loads html index page
func indexBytes() []byte {
	if _indexBytes == nil {
		var err error
		_indexBytes, err = ioutil.ReadFile("res/html/index.html")
		if err != nil {
			return nil
		}
	}
	return _indexBytes
}

// Index returns main page
func Index(writer http.ResponseWriter, req *http.Request) {
	logger, _, err := GetSettingsHandler(req)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	writer.Header().Set("Content-Type", "text/html")
	if _, err = writer.Write(indexBytes()); err != nil {
		logger.Error(err.Error())
	}
}

// RawInventory returns the raw inventory algorithm data
func RawInventory(writer http.ResponseWriter, req *http.Request) {
	// todo: fix

	//logger, _, err := GetSettingsHandler(req)
	//if err != nil {
	//	writer.WriteHeader(http.StatusBadRequest)
	//	return
	//}
	//
	//writer.Header().Set("Content-Type", "application/json")
	//
	//// todo
	//tags := inventory.GetRawInventory()
	//bytes, err := json.Marshal(tags)
	//if err != nil {
	//	logger.Error(err.Error())
	//	writer.WriteHeader(http.StatusInternalServerError)
	//	return
	//}
	//
	//if _, err = writer.Write(bytes); err != nil {
	//	logger.Error(err.Error())
	//	writer.WriteHeader(http.StatusInternalServerError)
	//}
}

// PingResponse sends pong back to client indicating service is up
func Ping(writer http.ResponseWriter, req *http.Request) {
	responseMessage := "pong"

	logger, _, err := GetSettingsHandler(req)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if err = WritePlainTextHTTPResponse(writer, responseMessage, http.StatusOK); err != nil {
		logger.Error(err.Error())
	}
}

// GetDevices gets device/reader list via EdgeX Core Metadata API
func GetDevices(writer http.ResponseWriter, req *http.Request) {
	logger, appSettings, err := GetSettingsHandler(req)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	logger.Info("Command to get the reader list called")
	deviceList, err := SendHTTPGetDevicesRequest(appSettings, client)
	if err != nil {
		responseMessage := err.Error()
		logger.Error(err.Error())
		if err = WritePlainTextHTTPResponse(writer, responseMessage, http.StatusInternalServerError); err != nil {
			logger.Error(err.Error())
		}
		return
	}
	if err = WriteJSONDeviceListHTTPResponse(writer, deviceList); err != nil {
		logger.Error(err.Error())
	}
}

// IssueReadOrStop sends start/stop reading command to the LLRP reader via EdgeX Core Command API
func IssueReadOrStop(writer http.ResponseWriter, req *http.Request) {
	logger, appSettings, err := GetSettingsHandler(req)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	issueCommandApi, err := GetAppSetting(appSettings, IssueCommandApi)
	if err != nil {
		responseMessage := err.Error()
		logger.Error(err.Error())
		if err = WritePlainTextHTTPResponse(writer, responseMessage, http.StatusInternalServerError); err != nil {
			logger.Error(err.Error())
		}
		return
	}

	vars := mux.Vars(req)
	readOrStop := vars[ReadOrStopKey]

	//Return back with error message if unable to parse command
	if !(readOrStop == StartReadingCommand || readOrStop == StopReadingCommand) {
		responseMessage := fmt.Sprintf("Bad request: unable to parse %v command", readOrStop)
		logger.Error(responseMessage)

		if err = WritePlainTextHTTPResponse(writer, responseMessage, http.StatusBadRequest); err != nil {
			logger.Error(err.Error())
		}
		return
	}

	// Get device list from EdgeX Core MetaData to send read/stop command
	deviceList, err := SendHTTPGetDevicesRequest(appSettings, client)
	if err != nil {
		responseMessage := err.Error()
		logger.Error(err.Error())
		if err = WritePlainTextHTTPResponse(writer, responseMessage, http.StatusInternalServerError); err != nil {
			logger.Error(err.Error())
		}
		return
	}

	var issueCommandErr bool
	var wg sync.WaitGroup
	wg.Add(len(deviceList))

	logger.Info(fmt.Sprintf("Sending %v command to all rfid registered devices", readOrStop))

	for _, deviceName := range deviceList {
		// GET core-command request to issue start/stop command to a device
		finalEndpoint := issueCommandApi + "/" + deviceName + "/command/" + readOrStop

		go func(finalEndpoint string) {

			defer wg.Done()

			if err := SendHTTPGETRequest(finalEndpoint, logger, client); err != nil {
				issueCommandErr = true
				logger.Error(err.Error())
			}
		}(finalEndpoint)
	}

	wg.Wait()

	if issueCommandErr {
		if err = WritePlainTextHTTPResponse(writer, "Error in sending command for one or more devices", http.StatusInternalServerError); err != nil {
			logger.Error(err.Error())
		}
		return
	}

	if err = WritePlainTextHTTPResponse(writer, "OK", http.StatusOK); err != nil {
		logger.Error(err.Error())
	}
}

// IssueBehavior sets behavior in the LLRP reader
func IssueBehavior(writer http.ResponseWriter, req *http.Request) {
	//TODO
}
