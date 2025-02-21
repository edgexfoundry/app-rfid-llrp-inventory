package llrp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/interfaces/mocks"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/dtos/responses"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func getTestingLogger() logger.LoggingClient {
	if testing.Verbose() {
		return logger.NewClient("test", "DEBUG")
	}

	return logger.NewMockClient()
}

func createMockCapabilities(t *testing.T, capJson string) map[string]interface{} {
	cap := make(map[string]interface{})
	err := json.Unmarshal([]byte(capJson), &cap)
	require.NoError(t, err)
	return cap
}

func TestNewReader(t *testing.T) {

	type testCase struct {
		testCaseName string
		deviceName   string
		respCode     int
		capabilities map[string]interface{}
	}

	penICap := createMockCapabilities(t, PENImpinjCap)
	penACap := createMockCapabilities(t, PENAlienCap)
	penZCap := createMockCapabilities(t, PENZebraCap)

	testCases := []testCase{
		{
			testCaseName: "Test New Reader Type for Device of Type PENImpinj",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			capabilities: penICap,
		},
		{
			testCaseName: "Test New Reader Type for Device of Type PENImpinj",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			capabilities: penACap,
		},
		{
			testCaseName: "Test New Reader Type for Device of Type PENZebra",
			deviceName:   "SpeedwayR-19-FE-16",
			respCode:     http.StatusOK,
			capabilities: penZCap,
		},
	}

	mockClient := &mocks.CommandClient{}
	deviceServiceClient := NewDSClient(mockClient, getTestingLogger())

	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			tcEvent := dtos.NewEvent("a", tc.deviceName, capReadingName)
			tcEvent.AddObjectReading(capReadingName, tc.capabilities)
			mockResp := responses.NewEventResponse("a", "b", tc.respCode, tcEvent)

			mockClient.On("IssueGetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mockResp, nil)
			mockClient.On("IssueSetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(common.BaseResponse{}, nil)

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

			tagReader, err := deviceServiceClient.NewReader(tc.deviceName)
			require.NoError(t, err)
			require.Equal(tt, reflect.TypeOf(tagReader), reflect.TypeOf(deviceType))
		})
	}

}

func TestGetCapabilities(t *testing.T) {

	type testCase struct {
		testCaseName  string
		deviceName    string
		capResponse   map[string]interface{}
		expectedCap   *GetReaderCapabilitiesResponse
		errorExpected bool
	}
	penICap := createMockCapabilities(t, PENImpinjCap)
	expectedCap := GetReaderCapabilitiesResponse{}
	err := json.Unmarshal([]byte(PENImpinjCap), &expectedCap)
	require.NoError(t, err)

	testCases := []testCase{
		{
			testCaseName:  "Test Unsuccessful HTTP GET Status Return",
			deviceName:    "SpeedwayR-19-FE-16",
			capResponse:   nil,
			expectedCap:   nil,
			errorExpected: true,
		},
		{
			testCaseName:  "Test Get Reader Capabilities Response",
			deviceName:    "SpeedwayR-19-FE-16",
			capResponse:   penICap,
			expectedCap:   &expectedCap,
			errorExpected: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {
			mockClient := &mocks.CommandClient{}
			if tc.errorExpected {
				mockResp := responses.NewEventResponse("a", "b", http.StatusBadRequest, dtos.Event{})
				mockClient.On("IssueGetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mockResp, errors.NewCommonEdgeXWrapper(fmt.Errorf("failed")))
			} else {
				tcEvent := dtos.NewEvent("a", tc.deviceName, capReadingName)
				tcEvent.AddObjectReading(capReadingName, tc.capResponse)
				mockResp := responses.NewEventResponse("a", "b", http.StatusOK, tcEvent)
				mockClient.On("IssueGetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&mockResp, nil)
			}

			deviceServiceClient := NewDSClient(mockClient, getTestingLogger())

			getReaderCapabilitiesResponse, err := deviceServiceClient.GetCapabilities(tc.deviceName)
			if tc.errorExpected {
				require.Error(tt, err)
				return
			}
			require.NoError(tt, err)
			require.NotNil(tt, getReaderCapabilitiesResponse)
			assert.Equal(tt, tc.expectedCap, getReaderCapabilitiesResponse)

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
		testCaseName  string
		deviceName    string
		fields        fields
		respCode      int
		errorExpected bool
	}

	testCases := []testCase{
		{
			testCaseName:  "Test Unsuccessful Config Set",
			deviceName:    "SpeedwayR-19-FE-16",
			fields:        fields{Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}}},
			respCode:      http.StatusBadRequest,
			errorExpected: true,
		},
		{
			testCaseName:  "Test Successful Config Set",
			deviceName:    "SpeedwayR-19-FE-16",
			fields:        fields{Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}}},
			respCode:      http.StatusOK,
			errorExpected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			mockClient := &mocks.CommandClient{}
			mockResp := common.NewBaseResponse("a", "b", tc.respCode)
			if tc.errorExpected {
				mockClient.On("IssueSetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockResp, errors.NewCommonEdgeXWrapper(fmt.Errorf("failed")))
			} else {
				mockClient.On("IssueSetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockResp, nil)
			}

			deviceServiceClient := NewDSClient(mockClient, getTestingLogger())

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

			err := deviceServiceClient.SetConfig(tc.deviceName, se)
			if tc.errorExpected {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
			}

		})
	}

}

func TestAddROSpec(t *testing.T) {
	type fields struct {
		ROSpec *ROSpec
	}
	type testCase struct {
		testCaseName  string
		deviceName    string
		fields        fields
		respCode      int
		errorExpected bool
	}

	testCases := []testCase{
		{
			testCaseName:  "Test Unsuccessful ROSpec Addition",
			deviceName:    "SpeedwayR-19-FE-16",
			fields:        fields{ROSpec: &ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}},
			respCode:      http.StatusBadRequest,
			errorExpected: true,
		},
		{
			testCaseName:  "Test Successful ROSpec Addition",
			deviceName:    "SpeedwayR-19-FE-16",
			fields:        fields{ROSpec: &ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}},
			respCode:      http.StatusOK,
			errorExpected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			mockClient := &mocks.CommandClient{}
			mockResp := common.NewBaseResponse("a", "b", tc.respCode)
			if tc.errorExpected {
				mockClient.On("IssueSetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockResp, errors.NewCommonEdgeXWrapper(fmt.Errorf("failed")))
			} else {
				mockClient.On("IssueSetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockResp, nil)
			}

			deviceServiceClient := NewDSClient(mockClient, getTestingLogger())

			err := deviceServiceClient.AddROSpec(tc.deviceName, tc.fields.ROSpec)
			if tc.errorExpected {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
			}
		})

	}

}

func TestModifyROSpecState(t *testing.T) {

	type testCase struct {
		testCaseName  string
		roCmd         string
		deviceName    string
		id            uint32
		respCode      int
		errorExpected bool
	}

	testCases := []testCase{
		{
			testCaseName:  "Test Enables ROSpec with the given ID on the given device",
			roCmd:         "enableCmd",
			deviceName:    "SpeedwayR-19-FE-16",
			id:            19865325,
			respCode:      http.StatusOK,
			errorExpected: false,
		},
		{
			testCaseName:  "Test Delete All ROSpec on a device",
			roCmd:         "deleteCmd",
			deviceName:    "SpeedwayR-19-FE-16",
			id:            0,
			respCode:      http.StatusOK,
			errorExpected: false,
		},
		{
			testCaseName:  "Test Unsuccessful Delete of All ROSpec on a device",
			roCmd:         "deleteCmd",
			deviceName:    "SpeedwayR-19-FE-16",
			id:            0,
			respCode:      http.StatusBadRequest,
			errorExpected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testCaseName, func(tt *testing.T) {

			mockClient := &mocks.CommandClient{}
			mockResp := common.NewBaseResponse("a", "b", tc.respCode)

			if tc.errorExpected {
				mockClient.On("IssueSetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockResp, errors.NewCommonEdgeXWrapper(fmt.Errorf("failed")))
			} else {
				mockClient.On("IssueSetCommandByName", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockResp, nil)
			}

			deviceServiceClient := NewDSClient(mockClient, getTestingLogger())

			err := deviceServiceClient.modifyROSpecState(tc.roCmd, tc.deviceName, tc.id)
			if tc.errorExpected {
				assert.Error(tt, err)
			} else {
				assert.NoError(tt, err)
			}
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
