package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
)

//Device struct
type Device struct {
	Name string `json:"name"`
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

// WriteJSONDeviceListHTTPResponse wrties an HTTP Response in JSON Format
func WriteJSONDeviceListHTTPResponse(w http.ResponseWriter, req *http.Request, content []string) error {
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(HTTPJSONDeviceListResponse{
		Content: content,
	})
	if err != nil {
		return err
	}

	return nil
}

// WritePlainTextHTTPResponse writes an HTTP Response in Plain Test Format
func WritePlainTextHTTPResponse(w http.ResponseWriter, req *http.Request, content string, statusCode int) error {
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, content)

	return nil
}
