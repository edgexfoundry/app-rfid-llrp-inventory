package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

// SendHTTPPUTRequest sends PUT Request to Edgex Core Command
func SendHTTPPUTRequest(readCommand string, putCommandEndpoint string, deviceName string, logger logger.LoggingClient, client *http.Client) error {
	finalEndpoint := putCommandEndpoint + "/" + deviceName + "/command/" + readCommand

	requestBody, err := json.Marshal(map[string]string{
		deviceName: readCommand,
	})
	if err != nil {
		return err
	}

	//Create New PUT request
	reqPUT, err := http.NewRequest(http.MethodPut, finalEndpoint, bytes.NewBuffer(requestBody))

	if err != nil {

		return err
	}

	//Set request header
	reqPUT.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := client.Do(reqPUT)

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
		return errors.Errorf("PUT to EdgeX Core failed with status %d; body: %q", resp.StatusCode, string(body))
	}

	logger.Info(fmt.Sprintf("Response from Edgex Core- %s", string(body)))
	return nil
}
