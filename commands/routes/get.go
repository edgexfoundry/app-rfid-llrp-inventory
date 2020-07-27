package routes

import (
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// SendHTTPGetDevicesRequest GET rest call to edgex-core-command to get the devices/readers list
func SendHTTPGetDevicesRequest(appSettings map[string]string, client *http.Client) ([]string, error) {
	coreCommandGetDevices, err := GetAppSetting(appSettings, CoreCommandGETDevices)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodGet, coreCommandGetDevices, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET call to edgex-core-command to get the readers failed: %d", resp.StatusCode)
	} else {
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		deviceList, err := GetDeviceList(respBody)
		if err != nil {
			return nil, errors.Errorf("Unable to parse device list from EdgeX: %s", err.Error())
		}
		if len(deviceList) == 0 {
			return nil, errors.Errorf("No devices registered")
		}
		return deviceList, nil
	}
}

// SendHTTPGETRequest sends GET Request to Edgex Core Command
func SendHTTPGETRequest(endpoint string, logger logger.LoggingClient, client *http.Client) error {
	logger.Debug(http.MethodGet + " " + endpoint)
	// create New GET request
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//Check & report for any error from EdgeX Core
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("GET to EdgeX Core failed with status %d; body: %q", resp.StatusCode, string(body))
	}

	logger.Debug("Response from Edgex Core: " + string(body))
	return nil

}
