package routes

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

// SendHTTPGetDeviceRequest send GET Device Request to Edgex Core Command
func SendHTTPGetDeviceRequest(appSettings map[string]string, client *http.Client) ([]string, error) {

	getCommandEndpoint := appSettings[CoreCommandGETDevicesCommandEndpoint]

	//Check for empty getCommandEndpoint
	if strings.TrimSpace(getCommandEndpoint) == "" {
		return nil, errors.Errorf("GET command Endpoint to EdgeX Core is nil")
	}

	//Create New Get request
	newReq, err := http.NewRequest(http.MethodGet, getCommandEndpoint, nil)

	if err != nil {

		return nil, err
	}

	//Set request header
	newReq.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(newReq)

	if err != nil {

		return nil, err
	}

	defer resp.Body.Close()

	//Read "DeviceLimit" Bytes from HTTP Response
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, DeviceLimit))
	if err != nil {
		return nil, err
	}

	//Send error response back to Client request
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET to EdgeX Core failed with status %d; body: %q", resp.StatusCode, string(body))
	}

	//Get device List by parsing response body
	deviceList, err := GetDeviceList(body)
	if err != nil {
		return nil, errors.Errorf("Error: Unable to parse device list from EdgeX- %s", err.Error())
	}

	//Return message if no devices available
	if len(deviceList) == 0 {
		//return an instance of deviceList to compare with nil check latter
		return []string{}, errors.Errorf("No Sensors Available")
	}

	return deviceList, nil
}
