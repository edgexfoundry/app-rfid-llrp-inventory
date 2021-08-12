package llrp

import (
	"encoding/json"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestingLogger() logger.LoggingClient {
	if testing.Verbose() {
		return logger.NewClientStdOut("test", false, "DEBUG")
	}

	return logger.NewMockClient()
}

func TestNewDSClient(t *testing.T) {

	assert := assert.New(t)
	type testCase struct {
		name    string
		hostURL url.URL
		client  *http.Client
		exp     interface{}
	}

	tests := []testCase{
		{
			name:    "Sample URL Test",
			hostURL: url.URL{Scheme: "https", Opaque: "", User: url.User("testUser"), Host: "testHost"},
			client:  http.DefaultClient,
			exp:     "https://testUser@testHost" + basePath,
		},
		{
			name:    "Default URL Test",
			hostURL: url.URL{},
			client:  http.DefaultClient,
			exp:     basePath,
		},
	}
	for _, ts := range tests {
		t.Run(ts.name, func(tt *testing.T) {
			dsClient := NewDSClient(&ts.hostURL, ts.client, getTestingLogger())
			assert.Equalf(ts.exp, dsClient.baseURL, "invalid value for baseURL: expected %+v, got %+v", ts.exp, dsClient.baseURL)

		})

	}
}

func TestGetDevices(t *testing.T) {

	type device struct{ Name string }

	type testCase struct {
		testCaseName string
		errMsg       string
		respCode     int
		devices      []device
	}

	testCases := []testCase{
		{
			testCaseName: "Test Unsuccessful HTTP GET Status Return",
			respCode:     http.StatusBadRequest,
			devices:      nil,
		},
		{
			testCaseName: "Test Get Device List",
			respCode:     http.StatusOK,
			devices:      []device{{Name: "SpeedwayR-19-FE-16"}, {Name: "SpeedwayR-19-BCclear-20"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.respCode)

				jsonData, err := json.Marshal(tc.devices)
				require.NoError(t, err)
				w.Write(jsonData)

			}

			s := httptest.NewServer(http.HandlerFunc(handler))

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			deviceServiceClient := NewDSClient(actualURL, s.Client(), getTestingLogger())

			deviceList, err := GetDevices(s.URL, deviceServiceClient.httpClient)
			if tc.respCode == http.StatusOK {
				assert.NotNil(tt, deviceList, "Expected device list to be not empty")
			} else {
				assert.NotNil(tt, err, "Encountered Error: %s", err)
			}

			s.Close()

		})

	}

}

func TestNewReader(t *testing.T) {

	type testCase struct {
		testCaseName string
		deviceName   string
		respCode     int
		capabilities string
	}

	testCases := []testCase{
		{
			testCaseName: "Test New Reader Type for Device of Type PENImpinj",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			capabilities: PENImpinjCap,
		},
		{
			testCaseName: "Test New Reader Type for Device of Type PENImpinj",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			capabilities: PENAlienCap,
		},
		{
			testCaseName: "Test New Reader Type for Device of Type PENZebra",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			capabilities: PENZebraCap,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.respCode)

				type Reading struct {
					Name, Value string
				}
				type edgexResp struct {
					Readings []Reading
				}
				if tc.capabilities != "" {
					resp := edgexResp{Readings: []Reading{{Name: capReadingName, Value: tc.capabilities}}}

					jsonData, err := json.Marshal(resp)
					require.NoError(t, err)
					w.Write(jsonData)
				}

			}

			s := httptest.NewServer(http.HandlerFunc(handler))

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			deviceServiceClient := NewDSClient(actualURL, s.Client(), getTestingLogger())

			getReaderCapabilitiesResponse, err := deviceServiceClient.GetCapabilities(tc.deviceName)
			require.NotNil(tt, getReaderCapabilitiesResponse, "err %s", err)

			var deviceType interface{}
			switch VendorPEN(getReaderCapabilitiesResponse.GeneralDeviceCapabilities.DeviceManufacturer) {
			case PENImpinj:
				deviceType, err = NewImpinjDevice(getReaderCapabilitiesResponse)
				require.NoError(t, err)
			default:
				deviceType, err = NewBasicDevice(getReaderCapabilitiesResponse)
				require.NoError(t, err)
			}

			tagReader, _ := deviceServiceClient.NewReader(tc.deviceName)

			require.Equal(tt, reflect.TypeOf(tagReader), reflect.TypeOf(deviceType))

			s.Close()

		})
	}

}

func TestGetCapabilities(t *testing.T) {

	type testCase struct {
		testCaseName string
		deviceName   string
		respCode     int
		respBody     string
		capabilities string
	}

	testCases := []testCase{
		{
			testCaseName: "Test Unsuccessful HTTP GET Status Return",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusBadRequest,
			respBody:     "",
			capabilities: "",
		},
		{
			testCaseName: "Test Json Parsing Error",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			respBody:     "{[]}",
			capabilities: "",
		},
		{
			testCaseName: "Test Empty Response Body",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			respBody:     "",
			capabilities: "",
		},
		{
			testCaseName: "Test Get Reader Capabilities Response",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			capabilities: PENImpinjCap,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.respCode)

				type Reading struct {
					Name, Value string
				}
				type edgexResp struct {
					Readings []Reading
				}
				if tc.capabilities != "" {
					resp := edgexResp{Readings: []Reading{{Name: capReadingName, Value: tc.capabilities}}}
					jsonData, err := json.Marshal(resp)
					require.NoError(t, err)
					w.Write(jsonData)
				} else {
					w.Write([]byte(tc.respBody))
				}
			}

			s := httptest.NewServer(http.HandlerFunc(handler))

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			deviceServiceClient := NewDSClient(actualURL, s.Client(), getTestingLogger())

			getReaderCapabilitiesResponse, errMsg := deviceServiceClient.GetCapabilities(tc.deviceName)

			if getReaderCapabilitiesResponse == nil {
				assert.NotNil(tt, errMsg, "Encountered Error: %s", errMsg)
			} else {
				assert.True(tt, ((getReaderCapabilitiesResponse != nil) && (errMsg == nil)), "Expected response %v for getReaderCapabilitiesResponse, received nil")
			}
			s.Close()

		})

	}

}

func TestSetConfig(t *testing.T) {
	type fields struct {
		ResetToFactoryDefaults      bool
		ReaderEventNotificationSpec *ReaderEventNotificationSpec
		AntennaProperties           []AntennaProperties
		AntennaConfigurations       []AntennaConfiguration
		ROReportSpec                *ROReportSpec
		AccessReportSpec            *AccessReportSpec
		KeepAliveSpec               *KeepAliveSpec
		GPOWriteData                []GPOWriteData
		GPIPortCurrentStates        []GPIPortCurrentState
		EventsAndReports            *EventsAndReports
		Custom                      []Custom
	}
	type testCase struct {
		testCaseName string
		deviceName   string
		fields       fields
		respCode     int
	}

	testCases := []testCase{
		{
			testCaseName: "Test Unsuccessful Config Set",
			deviceName:   "SpeedwayR-19-FE-16",
			fields:       fields{Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}}},
			respCode:     http.StatusBadRequest,
		},
		{
			testCaseName: "Test Successful Config Set",
			deviceName:   "SpeedwayR-19-FE-16",
			fields:       fields{Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}}},
			respCode:     http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.respCode)
			}

			s := httptest.NewServer(http.HandlerFunc(handler))

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			deviceServiceClient := NewDSClient(actualURL, s.Client(), getTestingLogger())

			se := &SetReaderConfig{
				ResetToFactoryDefaults:      tc.fields.ResetToFactoryDefaults,
				ReaderEventNotificationSpec: tc.fields.ReaderEventNotificationSpec,
				AntennaProperties:           tc.fields.AntennaProperties,
				AntennaConfigurations:       tc.fields.AntennaConfigurations,
				ROReportSpec:                tc.fields.ROReportSpec,
				AccessReportSpec:            tc.fields.AccessReportSpec,
				KeepAliveSpec:               tc.fields.KeepAliveSpec,
				GPOWriteData:                tc.fields.GPOWriteData,
				GPIPortCurrentStates:        tc.fields.GPIPortCurrentStates,
				EventsAndReports:            tc.fields.EventsAndReports,
				Custom:                      tc.fields.Custom,
			}

			errMsg := deviceServiceClient.SetConfig(tc.deviceName, se)
			if tc.respCode == http.StatusOK {
				assert.Nil(tt, errMsg, "Encountered Error: %s", errMsg)
			} else {
				assert.NotNil(tt, errMsg, "Encountered Error: %s", errMsg)
			}

			s.Close()

		})

	}

}

func TestAddROSpec(t *testing.T) {
	type fields struct {
		ROSpec ROSpec
	}
	type testCase struct {
		testCaseName string
		deviceName   string
		fields       fields
		respCode     int
	}

	testCases := []testCase{
		{
			testCaseName: "Test Unsuccessful ROSpec Addition",
			deviceName:   "SpeedwayR-19-FE-16",
			fields:       fields{ROSpec: ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}},
			respCode:     http.StatusBadRequest,
		},
		{
			testCaseName: "Test Successful ROSpec Addition",
			deviceName:   "SpeedwayR-19-FE-16",
			fields:       fields{ROSpec: ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}},
			respCode:     http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.respCode)
			}

			s := httptest.NewServer(http.HandlerFunc(handler))

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			deviceServiceClient := NewDSClient(actualURL, s.Client(), getTestingLogger())

			errMsg := deviceServiceClient.AddROSpec(tc.deviceName, &tc.fields.ROSpec)
			if tc.respCode == http.StatusOK {
				assert.Nil(tt, errMsg, "Encountered Error: %s", errMsg)
			} else {
				assert.NotNil(tt, errMsg, "Encountered Error: %s", errMsg)
			}

			s.Close()

		})

	}

}

func TestModifyROSpecState(t *testing.T) {

	type testCase struct {
		testCaseName string
		roCmd        string
		deviceName   string
		id           uint32
		respCode     int
	}

	testCases := []testCase{
		{
			testCaseName: "Test Enables ROSpec with the given ID on the given device",
			roCmd:        "enableCmd",
			deviceName:   "SpeedwayR-19-FE-16",
			id:           19865325,
			respCode:     http.StatusOK,
		},
		{
			testCaseName: "Test Delete All ROSpec on a device",
			roCmd:        "deleteCmd",
			deviceName:   "SpeedwayR-19-FE-16",
			id:           0,
			respCode:     http.StatusOK,
		},
		{
			testCaseName: "Test Unsuccessful Delete of All ROSpec on a device",
			roCmd:        "deleteCmd",
			deviceName:   "SpeedwayR-19-FE-16",
			id:           0,
			respCode:     http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			handler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.respCode)
			}

			s := httptest.NewServer(http.HandlerFunc(handler))

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			deviceServiceClient := NewDSClient(actualURL, s.Client(), getTestingLogger())

			errMsg := deviceServiceClient.modifyROSpecState(tc.roCmd, tc.deviceName, tc.id)
			if tc.respCode == http.StatusOK {
				assert.Nil(tt, errMsg, "Encountered Error: %s", errMsg)
			} else {
				assert.NotNil(tt, errMsg, "Encountered Error: %s", errMsg)
			}

			s.Close()

		})

	}

}

func TestPut(t *testing.T) {
	type testCase struct {
		testCaseName   string
		path           string
		data           []byte
		serverShutDown bool
		respCode       int
	}

	testCases := []testCase{
		{
			testCaseName: "Test Unsuccessful HTTP PUT Status",
			path:         "SpeedwayR-19-FE-16" + configDevCmd,
			respCode:     http.StatusBadRequest,
		},
		{
			testCaseName: "Test Successful HTTP PUT Statud",
			path:         "SpeedwayR-19-FE-16" + configDevCmd,
			respCode:     http.StatusOK,
		},
		{
			testCaseName:   "Test Server Shutdown",
			path:           "SpeedwayR-19-FE-16" + configDevCmd,
			serverShutDown: true,
			respCode:       http.StatusInternalServerError,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			handler := func(w http.ResponseWriter, r *http.Request) {
				r.Method = "PUT"
				w.WriteHeader(tc.respCode)
			}

			s := httptest.NewServer(http.HandlerFunc(handler))

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			deviceServiceClient := NewDSClient(actualURL, s.Client(), getTestingLogger())

			if tc.serverShutDown {
				s.Close()
			}

			err = deviceServiceClient.put(tc.path, tc.data)
			if tc.respCode != http.StatusOK {
				assert.NotNil(tt, err, "Encountered Error: %s", err)
			} else {
				assert.Nil(tt, err, "Encountered Error: %s", err)
			}

			s.Close()

		})

	}

}

func TestTryGet(t *testing.T) {
	testCases := []struct {
		name           string
		path           string
		spoofCode      int
		spoofCodes     []int
		expectedTries  int
		expectedStatus int
		spoofConnErr   bool
		expectErr      bool
	}{
		{
			name:           "200 ok - first try",
			spoofCode:      http.StatusOK,
			expectedStatus: http.StatusOK,
			expectedTries:  1,
		},
		{
			name:           "204 no content - first try",
			spoofCode:      http.StatusNoContent,
			expectedStatus: http.StatusNoContent,
			expectedTries:  1, // StatusNoContent (204) is a success code, so only 1 try
		},
		{
			name:           "202 accepted - first try",
			spoofCode:      http.StatusAccepted,
			expectedStatus: http.StatusAccepted,
			expectedTries:  1, // StatusAccepted (202) is a success code, so only 1 try
		},
		{
			name:           "http 423, retry 200 ok",
			spoofCodes:     []int{http.StatusLocked, http.StatusOK},
			expectedStatus: http.StatusOK,
			expectedTries:  2,
		},
		{
			name: "http MovedPermanently errors",
			// return StatusMovedPermanently, but do not set location header, causes handler error
			spoofCode:     http.StatusMovedPermanently,
			expectErr:     true,
			expectedTries: maxTries,
		},
		{
			name: "http 3xx is not success",
			// 300, 303, 206
			spoofCodes:     []int{http.StatusMultipleChoices, http.StatusSeeOther, http.StatusPartialContent},
			expectedStatus: http.StatusPartialContent, // 206
			expectedTries:  3,
		},
		{
			name:           "http 502, http 504, retry 200 ok",
			spoofCodes:     []int{http.StatusBadGateway, http.StatusGatewayTimeout, http.StatusOK},
			expectedStatus: http.StatusOK,
			expectedTries:  3,
		},
		{
			name:           "fail http 423 every try",
			spoofCode:      http.StatusLocked,
			expectedTries:  maxTries,
			expectedStatus: http.StatusLocked,
		},
		{
			name:          "invalid url error",
			path:          string([]byte{0x7f}), // ASCII control character
			expectErr:     true,
			expectedTries: 0, // 0 tries as the request object is invalid
		},
		{
			name:          "EOF connection error",
			spoofConnErr:  true,
			expectErr:     true,
			expectedTries: maxTries,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// define the pointer here, so that it can be used in the handler below
			var s *httptest.Server
			var tries int

			handler := func(w http.ResponseWriter, r *http.Request) {
				tries++
				if tc.spoofConnErr {
					// forcefully close the client connection to spoof an error
					s.CloseClientConnections()
				} else if len(tc.spoofCodes) >= tries {
					w.WriteHeader(tc.spoofCodes[tries-1])
				} else if tc.spoofCode != 0 {
					w.WriteHeader(tc.spoofCode)
				} else {
					assert.FailNow(t, "invalid test case configuration")
				}
			}

			// spin up a fake test server
			s = httptest.NewServer(http.HandlerFunc(handler))
			defer s.Close()

			actualURL, err := url.Parse(s.URL)
			require.NoError(t, err)
			ds := NewDSClient(actualURL, s.Client(), getTestingLogger())

			resp, err := ds.tryGet(tc.path)
			if err == nil { // if err is nil, we must close body
				defer resp.Body.Close()
			}
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tc.expectedStatus != 0 {
					assert.Equal(t, tc.expectedStatus, resp.StatusCode)
				}
			}
			assert.Equal(t, tc.expectedTries, tries)
		})
	}
}

const PENImpinjCap = `{
	"LLRPStatus": {
		"Status": 0,
		"ErrorDescription": "",
		"FieldError": null,
		"ParameterError": null
	},
	"GeneralDeviceCapabilities": {
		"MaxSupportedAntennas": 4,
		"CanSetAntennaProperties": false,
		"HasUTCClock": true,
		"DeviceManufacturer": 25882,
		"Model": 2001002,
		"FirmwareVersion": "5.14.0.240",
		"ReceiveSensitivities": [
			{
				"Index": 1,
				"ReceiveSensitivity": 0
			},
			{
				"Index": 2,
				"ReceiveSensitivity": 10
			}
		],
		"PerAntennaReceiveSensitivityRanges": null,
		"GPIOCapabilities": {
			"NumGPIs": 4,
			"NumGPOs": 4
		},
		"PerAntennaAirProtocols": [
			{
				"AntennaID": 1,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 2,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 3,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 4,
				"AirProtocolIDs": "AQ=="
			}
		],
		"MaximumReceiveSensitivity": null
	},
	"LLRPCapabilities": {
		"CanDoRFSurvey": false,
		"CanReportBufferFillWarning": true,
		"SupportsClientRequestOpSpec": false,
		"CanDoTagInventoryStateAwareSingulation": false,
		"SupportsEventsAndReportHolding": true,
		"MaxPriorityLevelSupported": 1,
		"ClientRequestedOpSpecTimeout": 0,
		"MaxROSpecs": 1,
		"MaxSpecsPerROSpec": 32,
		"MaxInventoryParameterSpecsPerAISpec": 1,
		"MaxAccessSpecs": 1508,
		"MaxOpSpecsPerAccessSpec": 8
	},
	"RegulatoryCapabilities": {
		"CountryCode": 840,
		"CommunicationsStandard": 1,
		"UHFBandCapabilities": {
			"TransmitPowerLevels": [
				{
					"Index": 1,
					"TransmitPowerValue": 1000
				}
			],
			"FrequencyInformation": {
				"Hopping": true,
				"FrequencyHopTables": [
					{
						"HopTableID": 1,
						"Frequencies": [
							909250,
							908250,
							925750,
							911250
							 ]
					}
				],
				"FixedFrequencyTable": null
			},
			"C1G2RFModes": {
				"UHFC1G2RFModeTableEntries": [
					{
						"ModeID": 0,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 1,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 2,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 2,
						"ForwardLinkModulation": 0,
						"SpectralMask": 3,
						"BackscatterDataRate": 274000,
						"PIERatio": 2000,
						"MinTariTime": 20000,
						"MaxTariTime": 20000,
						"StepTariTime": 0
					}

				]
			},
			"RFSurveyFrequencyCapabilities": null
		},
		"Custom": null
	},
	"C1G2LLRPCapabilities": {
		"SupportsBlockErase": false,
		"SupportsBlockWrite": true,
		"SupportsBlockPermalock": false,
		"SupportsTagRecommissioning": false,
		"SupportsUMIMethod2": false,
		"SupportsXPC": false,
		"MaxSelectFiltersPerQuery": 2
	},
	"Custom": null
}`

const PENAlienCap = `{
	"LLRPStatus": {
		"Status": 0,
		"ErrorDescription": "",
		"FieldError": null,
		"ParameterError": null
	},
	"GeneralDeviceCapabilities": {
		"MaxSupportedAntennas": 4,
		"CanSetAntennaProperties": false,
		"HasUTCClock": true,
		"DeviceManufacturer": 17996,
		"Model": 2001002,
		"FirmwareVersion": "5.14.0.240",
		"ReceiveSensitivities": [
			{
				"Index": 1,
				"ReceiveSensitivity": 0
			},
			{
				"Index": 2,
				"ReceiveSensitivity": 10
			}
		],
		"PerAntennaReceiveSensitivityRanges": null,
		"GPIOCapabilities": {
			"NumGPIs": 4,
			"NumGPOs": 4
		},
		"PerAntennaAirProtocols": [
			{
				"AntennaID": 1,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 2,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 3,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 4,
				"AirProtocolIDs": "AQ=="
			}
		],
		"MaximumReceiveSensitivity": null
	},
	"LLRPCapabilities": {
		"CanDoRFSurvey": false,
		"CanReportBufferFillWarning": true,
		"SupportsClientRequestOpSpec": false,
		"CanDoTagInventoryStateAwareSingulation": false,
		"SupportsEventsAndReportHolding": true,
		"MaxPriorityLevelSupported": 1,
		"ClientRequestedOpSpecTimeout": 0,
		"MaxROSpecs": 1,
		"MaxSpecsPerROSpec": 32,
		"MaxInventoryParameterSpecsPerAISpec": 1,
		"MaxAccessSpecs": 1508,
		"MaxOpSpecsPerAccessSpec": 8
	},
	"RegulatoryCapabilities": {
		"CountryCode": 840,
		"CommunicationsStandard": 1,
		"UHFBandCapabilities": {
			"TransmitPowerLevels": [
				{
					"Index": 1,
					"TransmitPowerValue": 1000
				}
			],
			"FrequencyInformation": {
				"Hopping": true,
				"FrequencyHopTables": [
					{
						"HopTableID": 1,
						"Frequencies": [
							909250,
							908250,
							925750,
							911250
							 ]
					}
				],
				"FixedFrequencyTable": null
			},
			"C1G2RFModes": {
				"UHFC1G2RFModeTableEntries": [
					{
						"ModeID": 0,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 1,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 2,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 2,
						"ForwardLinkModulation": 0,
						"SpectralMask": 3,
						"BackscatterDataRate": 274000,
						"PIERatio": 2000,
						"MinTariTime": 20000,
						"MaxTariTime": 20000,
						"StepTariTime": 0
					}

				]
			},
			"RFSurveyFrequencyCapabilities": null
		},
		"Custom": null
	},
	"C1G2LLRPCapabilities": {
		"SupportsBlockErase": false,
		"SupportsBlockWrite": true,
		"SupportsBlockPermalock": false,
		"SupportsTagRecommissioning": false,
		"SupportsUMIMethod2": false,
		"SupportsXPC": false,
		"MaxSelectFiltersPerQuery": 2
	},
	"Custom": null
}`

const PENZebraCap = `{
	"LLRPStatus": {
		"Status": 0,
		"ErrorDescription": "",
		"FieldError": null,
		"ParameterError": null
	},
	"GeneralDeviceCapabilities": {
		"MaxSupportedAntennas": 4,
		"CanSetAntennaProperties": false,
		"HasUTCClock": true,
		"DeviceManufacturer": 10642,
		"Model": 2001002,
		"FirmwareVersion": "5.14.0.240",
		"ReceiveSensitivities": [
			{
				"Index": 1,
				"ReceiveSensitivity": 0
			},
			{
				"Index": 2,
				"ReceiveSensitivity": 10
			}
		],
		"PerAntennaReceiveSensitivityRanges": null,
		"GPIOCapabilities": {
			"NumGPIs": 4,
			"NumGPOs": 4
		},
		"PerAntennaAirProtocols": [
			{
				"AntennaID": 1,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 2,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 3,
				"AirProtocolIDs": "AQ=="
			},
			{
				"AntennaID": 4,
				"AirProtocolIDs": "AQ=="
			}
		],
		"MaximumReceiveSensitivity": null
	},
	"LLRPCapabilities": {
		"CanDoRFSurvey": false,
		"CanReportBufferFillWarning": true,
		"SupportsClientRequestOpSpec": false,
		"CanDoTagInventoryStateAwareSingulation": false,
		"SupportsEventsAndReportHolding": true,
		"MaxPriorityLevelSupported": 1,
		"ClientRequestedOpSpecTimeout": 0,
		"MaxROSpecs": 1,
		"MaxSpecsPerROSpec": 32,
		"MaxInventoryParameterSpecsPerAISpec": 1,
		"MaxAccessSpecs": 1508,
		"MaxOpSpecsPerAccessSpec": 8
	},
	"RegulatoryCapabilities": {
		"CountryCode": 840,
		"CommunicationsStandard": 1,
		"UHFBandCapabilities": {
			"TransmitPowerLevels": [
				{
					"Index": 1,
					"TransmitPowerValue": 1000
				}
			],
			"FrequencyInformation": {
				"Hopping": true,
				"FrequencyHopTables": [
					{
						"HopTableID": 1,
						"Frequencies": [
							909250,
							908250,
							925750,
							911250
							 ]
					}
				],
				"FixedFrequencyTable": null
			},
			"C1G2RFModes": {
				"UHFC1G2RFModeTableEntries": [
					{
						"ModeID": 0,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 0,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 1,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 1,
						"ForwardLinkModulation": 2,
						"SpectralMask": 2,
						"BackscatterDataRate": 640000,
						"PIERatio": 1500,
						"MinTariTime": 6250,
						"MaxTariTime": 6250,
						"StepTariTime": 0
					},
					{
						"ModeID": 2,
						"DivideRatio": 1,
						"IsEPCHagConformant": false,
						"Modulation": 2,
						"ForwardLinkModulation": 0,
						"SpectralMask": 3,
						"BackscatterDataRate": 274000,
						"PIERatio": 2000,
						"MinTariTime": 20000,
						"MaxTariTime": 20000,
						"StepTariTime": 0
					}

				]
			},
			"RFSurveyFrequencyCapabilities": null
		},
		"Custom": null
	},
	"C1G2LLRPCapabilities": {
		"SupportsBlockErase": false,
		"SupportsBlockWrite": true,
		"SupportsBlockPermalock": false,
		"SupportsTagRecommissioning": false,
		"SupportsUMIMethod2": false,
		"SupportsXPC": false,
		"MaxSelectFiltersPerQuery": 2
	},
	"Custom": null
}`
