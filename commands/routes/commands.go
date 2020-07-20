package routes

import (
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
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
	if err = WriteHtmlHttpResponse(writer, indexBytes()); err != nil {
		logger.Error(err.Error())
	}
}

// PingResponse sends pong back to client indicating service is up
func PingResponse(writer http.ResponseWriter, req *http.Request) {
	responseMessage := "pong"
	logger, _, err := GetSettingsHandler(req)

	if err != nil {
		if err = WritePlainTextHTTPResponse(writer, "", http.StatusInternalServerError); err != nil {
			logger.Error(err.Error())
		}
	} else {
		if err = WritePlainTextHTTPResponse(writer, responseMessage, http.StatusOK); err != nil {
			logger.Error(err.Error())
		}
	}
}

// GetDevicesCommand gets device/reader list via EdgeX Core Command API
func GetDevicesCommand(writer http.ResponseWriter, req *http.Request) {
	logger, appSettings, err := GetSettingsHandler(req)
	if err != nil {
		if err = WritePlainTextHTTPResponse(writer, "", http.StatusInternalServerError); err != nil {
			logger.Error(err.Error())
		}
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
	} else {
		//Send list of registered rfid devices to Client request
		if err = WriteJSONDeviceListHTTPResponse(writer, deviceList); err != nil {
			logger.Error(err.Error())
		}
	}
}

// IssueReadCommand sends start/stop reading command via EdgeX Core Command API
func IssueReadCommand(writer http.ResponseWriter, req *http.Request) {
	//Initialize response parameters
	responseMessage := ""
	httpResponseCode := http.StatusOK

	logger, appSettings, err := GetSettingsHandler(req)

	if err != nil {
		responseMessage = http.StatusText(http.StatusInternalServerError)
		httpResponseCode = http.StatusInternalServerError
		if logger != nil {
			logger.Error(err.Error())
		} else {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		if werr := WritePlainTextHTTPResponse(writer, responseMessage, httpResponseCode); werr != nil {
			logger.Error(werr.Error())
		}
		return
	}

	putCommandEndpoint := appSettings[CoreCommandPUTDevice]
	//Check for empty putCommandEndpoint
	if strings.TrimSpace(putCommandEndpoint) == "" {
		responseMessage = http.StatusText(http.StatusInternalServerError)
		httpResponseCode = http.StatusInternalServerError
		logger.Error("PUT command Endpoint to EdgeX Core is nil")
		if werr := WritePlainTextHTTPResponse(writer, responseMessage, httpResponseCode); werr != nil {
			logger.Error(werr.Error())
		}
		return
	}

	vars := mux.Vars(req)
	readCommand := vars[ReadCommandKey]

	logger.Info(fmt.Sprintf("readCommand to be sent to registered devices is %s", readCommand))

	//Return back with error message if unable to parse Read Command
	if !(readCommand == StartReadingCommand || readCommand == StopReadingCommand) {

		responseMessage = fmt.Sprintf("Unable to parse %v Command", readCommand)
		httpResponseCode = http.StatusBadRequest
		logger.Error(responseMessage)

		//Send response back to Client request
		if werr := WritePlainTextHTTPResponse(writer, responseMessage, httpResponseCode); werr != nil {
			logger.Error(werr.Error())
		}
		return

	}

	// todo: this should not be done here
	// Get Device List from EdgeX Core Command
	deviceList, err := SendHTTPGetDevicesRequest(appSettings, client)
	if err != nil {
		//Log the actual error & display response message to Client as "Internal Server Error"
		if deviceList != nil {
			responseMessage = err.Error()
		} else {
			responseMessage = http.StatusText(http.StatusInternalServerError)
		}
		httpResponseCode = http.StatusInternalServerError
		logger.Error(err.Error())

		if werr := WritePlainTextHTTPResponse(writer, responseMessage, httpResponseCode); werr != nil {
			logger.Error(werr.Error())
		}
		return
	}

	//Empty device List check done in SendHTTPGetRequest function, error logged in 122 & return back
	deviceListLength := len(deviceList)

	//sendErrs track any unsuccessful PUT request to EdgeX Core Command
	sendErrs := make([]bool, deviceListLength)

	//Create & Add devices count into waitgroup
	var wg sync.WaitGroup
	wg.Add(deviceListLength)

	logger.Info(fmt.Sprintf("Sending %v Command to all rfid registered devices", readCommand))

	for i, deviceName := range deviceList {
		go func(i int, deviceName string) {

			//Delete from waitgroup
			defer wg.Done()

			//PUT request to device-deviceName via EdgeX Core Command
			finalEndpoint := putCommandEndpoint + "/" + deviceName + "/command/" + readCommand
			err := SendHTTPGETRequest(finalEndpoint, logger, client)
			if err != nil {
				sendErrs[i] = true
				logger.Error(fmt.Sprintf("Error sending %v Command to device %v via EdgeX Core-Command", readCommand, deviceName))
			}
		}(i, deviceName)
	}

	//Wait until all in waitgroup are executed
	wg.Wait()

	//Successful Response back to Client Request
	responseMessage = "OK"
	for _, errYes := range sendErrs {
		if errYes {
			//Unsuccessful Response back to Client Request
			httpResponseCode = http.StatusInternalServerError
			responseMessage = fmt.Sprintf("Unsuccessful in sending %v Command", readCommand)
			break
		}

	}

	//Send response back to Client requent
	if werr := WritePlainTextHTTPResponse(writer, responseMessage, httpResponseCode); werr != nil {
		logger.Error(werr.Error())
	}
}

// IssueBehaviorCommand sends command to set/apply behavior command
func IssueBehaviorCommand(writer http.ResponseWriter, req *http.Request) {
	//TODO
}
