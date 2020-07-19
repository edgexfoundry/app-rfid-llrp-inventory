package routes

import (
	"path/filepath"

	"github.com/pelletier/go-toml"

	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
)

var appSettings map[string]string

func initialize() map[string]SettingsHandler {

	//var appSettings map[string]string
	var newLogger logger.LoggingClient
	newLogger = logger.NewClient("test", false, "", "DEBUG")

	fileLoc, _ := filepath.Abs("../../rfid-commands-service/res/commands.toml")

	config, err := toml.LoadFile(fileLoc)
	if err != nil {
		newLogger.Error(err.Error())
	}

	if putDevicesNameCommandEndpoint := config.Get("ApplicationSettings.CoreCommandPUTDevicesNameCommandEndpoint"); putDevicesNameCommandEndpoint == nil {
		newLogger.Error("***Error: error in reading CoreCommandPUTDevicesNameCommandEndpoint from commands.toml file***")
		return nil
	}

	if getDevicesNameCommandEndpoint := config.Get("ApplicationSettings.CoreCommandGETDevicesCommandEndpoint"); getDevicesNameCommandEndpoint == nil {
		newLogger.Error("***Error: error in reading CoreCommandGETDevicesCommandEndpoint from commands.toml file***")
		return nil
	}
	appSettings = map[string]string{CoreCommandPUTDevice: config.Get("ApplicationSettings.CoreCommandPUTDevicesNameCommandEndpoint").(string),
		CoreCommandGETDevices: config.Get("ApplicationSettings.CoreCommandGETDevicesCommandEndpoint").(string),
	}

	settingsHandlerVar := SettingsHandler{Logger: newLogger, AppSettings: appSettings}
	settingsMap := map[string]SettingsHandler{SettingsHandlerKey: settingsHandlerVar}

	return settingsMap
}

/*
func TestPingResponse(t *testing.T) {

	tests := []struct {
		name               string
		uRL                string
		method             string
		returnValue        string
		expectedStatusCode int
		settingsMap        map[string]SettingsHandler
	}{
		{"pingTestPass", "http://localhost:49993/ping", "GET", "pong", http.StatusOK, initialize()},
		{"pingTestFail", "http://localhost:49993/ping", "GET", http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError, nil},
	}

	for _, test := range tests {
		currentTest := test
		t.Run(currentTest.name, func(t *testing.T) {

			req := httptest.NewRequest(currentTest.method, currentTest.uRL, nil)

			ctx := context.WithValue(req.Context(), SettingsMapKey, currentTest.settingsMap)

			w := httptest.NewRecorder()
			PingResponse(w, req.WithContext(ctx))
			resp := w.Result()
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				t.Errorf("****************Error: error in reading the response body*************")
			}

			require.Equal(t, currentTest.expectedStatusCode, resp.StatusCode, "invalid status code")
			require.Equal(t, currentTest.returnValue, string(body), "invalid return string")

		})
	}
}

func TestGetSensorsCommand(t *testing.T) {

	tests := []struct {
		name               string
		uRL                string
		method             string
		returnValue        interface{}
		expectedStatusCode int
		settingsMap        map[string]SettingsHandler
	}{
		{"getNSensorsTest", "http://localhost:49993/command/sensors", "GET", nil, http.StatusOK, initialize()},
		{"getZeroSensorsTest", "http://localhost:49993/command/sensors", "GET", "No Sensors Available", http.StatusInternalServerError, initialize()},
		{"getSensorsTestFail", "http://localhost:49993/command/sensors", "GET", http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError, nil},
	}

	for _, test := range tests {
		currentTest := test
		t.Run(currentTest.name, func(t *testing.T) {

			req := httptest.NewRequest(currentTest.method, currentTest.uRL, nil)

			ctx := context.WithValue(req.Context(), SettingsMapKey, currentTest.settingsMap)

			w := httptest.NewRecorder()
			GetSensorsCommand(w, req.WithContext(ctx))
			resp := w.Result()
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				t.Errorf("****************Error: error in reading the response body*************")
			}

			//Test for return value - error string or device count
			if currentTest.returnValue != nil {
				//Test if error message returned is correct when either settingsMap is nil or device list is empty
				//require.Equal(t, currentTest.expectedStatusCode, resp.StatusCode, "invalid return message")
				require.Equal(t, currentTest.returnValue, string(body), "invalid return message")
			} else {

				deviceList := strings.Split(string(body), "\n")
				if deviceList == nil {
					t.Errorf("****************Error: error in getting device list from EdgeX*************")
				}
				//Test if device list is returned successfully from EdgeX
				if currentTest.expectedStatusCode == resp.StatusCode {
					assert.GreaterOrEqual(t, len(deviceList), 1)
				} else {
					t.Errorf("****************Error: No devices available*************")
				}
			}

		})
	}
}*/

/*func TestIssueReadCommand(t *testing.T) {

	tests := []struct {
		name               string
		uRL                string
		readCommand        string
		method             string
		returnValue        string
		expectedStatusCode int
		settingsMap        map[string]SettingsHandler
	}{
		{"issueSTARTReadCommandTestSuccessful", "http://localhost:49993/command/readings/readCommand", StartReadingCommand, "PUT", "Successfully sent START_READING Command  to all registered rfid devices via EdgeX Core-Command", http.StatusOK, initialize()},
		{"issueSTOPReadCommandTestSuccessful", "http://localhost:49993/command/readings/readCommand", StopReadingCommand, "PUT", "Successfully sent STOP_READING Command  to all registered rfid devices via EdgeX Core-Command", http.StatusOK, initialize()},
		{"issueSTARTReadCommandTestUnsuccessful", "http://localhost:49993/command/readings/readCommand", StartReadingCommand, "PUT", "Unsuccessful in sending START_READING Command", http.StatusInternalServerError, initialize()},
		{"issueSTOPReadCommandTestUnsuccessful", "http://localhost:49993/command/readings/readCommand", StopReadingCommand, "PUT", "Unsuccessful in sending STOP_READING Command", http.StatusInternalServerError, initialize()},
		{"issueSTARTReadCommandWithZeroDeviceTest", "http://localhost:49993/command/readings/readCommand", StartReadingCommand, "PUT", "No Sensors Available", http.StatusInternalServerError, initialize()},
		{"issueSTOPReadCommandWithZeroDeviceTest", "http://localhost:49993/command/readings/readCommand", StopReadingCommand, "PUT", "No Sensors Available", http.StatusInternalServerError, initialize()},
		{"issueSTARTReadCommandTestFail", "http://localhost:49993/command/readings/readCommand", StartReadingCommand, "PUT", http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError, nil},
		{"issueSTOPReadCommandTestFail", "http://localhost:49993/command/readings/readCommand", StopReadingCommand, "PUT", http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError, nil},
	}

	for _, test := range tests {
		currentTest := test
		t.Run(currentTest.name, func(t *testing.T) {

			req := httptest.NewRequest(currentTest.method, currentTest.uRL, nil)
			req = mux.SetURLVars(req, map[string]string{"readCommand": currentTest.readCommand})

			ctx := context.WithValue(req.Context(), SettingsMapKey, currentTest.settingsMap)

			w := httptest.NewRecorder()
			IssueReadCommand(w, req.WithContext(ctx))
			resp := w.Result()
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)

			if err != nil {
				t.Errorf("****************Error: error in reading the response body*************")
			}

			if resp.StatusCode == http.StatusOK {
				require.Equal(t, currentTest.returnValue, string(body), "invalid return message")
			} else if resp.StatusCode == http.StatusInternalServerError {
				require.Equal(t, currentTest.returnValue, string(body), "invalid return message")
			} else {
				t.Errorf("****************Error: error message sent back to client, check log for error message*************")
			}

		})
	}
}*/
