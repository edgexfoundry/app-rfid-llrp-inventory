package routes

import (
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

// SendHTTPGetDeviceRequest GET rest call to edgex-core-command to get the devices/readers list
func SendHTTPGetDevicesRequest(appSettings map[string]string, client *http.Client) ([]string, error) {
	coreCommandGetDevices, ok := appSettings[CoreCommandGETDevices]
	if !ok || coreCommandGetDevices == "" {
		return nil, errors.Errorf("App settings for edgex-core-command api to get readers/devices is not set")
	}

	req, err := http.NewRequest(http.MethodGet, coreCommandGetDevices, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
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
