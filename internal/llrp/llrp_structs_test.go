package llrp

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSupportedVersionResponse_Type(t *testing.T) {
	type fields struct {
		CurrentVersion      VersionNum
		MaxSupportedVersion VersionNum
		LLRPStatus          LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK",
			fields: fields{CurrentVersion: Version1_0_1, MaxSupportedVersion: Version1_1, LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgGetSupportedVersionResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetSupportedVersionResponse{
				CurrentVersion:      tt.fields.CurrentVersion,
				MaxSupportedVersion: tt.fields.MaxSupportedVersion,
				LLRPStatus:          tt.fields.LLRPStatus,
			}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetSupportedVersionResponse_Status(t *testing.T) {
	type fields struct {
		CurrentVersion      VersionNum
		MaxSupportedVersion VersionNum
		LLRPStatus          LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{CurrentVersion: Version1_0_1, MaxSupportedVersion: Version1_1, LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{CurrentVersion: Version1_0_1, MaxSupportedVersion: Version1_1, LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{CurrentVersion: Version1_0_1, MaxSupportedVersion: Version1_1, LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GetSupportedVersionResponse{
				CurrentVersion:      tt.fields.CurrentVersion,
				MaxSupportedVersion: tt.fields.MaxSupportedVersion,
				LLRPStatus:          tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestSetProtocolVersion_Type(t *testing.T) {
	type fields struct {
		TargetVersion VersionNum
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK",
			fields: fields{TargetVersion: Version1_1},
			want:   MsgSetProtocolVersion,
		},
		{
			name:   "OK - unknown version",
			fields: fields{TargetVersion: versionUnknown},
			want:   MsgSetProtocolVersion,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &SetProtocolVersion{
				TargetVersion: tt.fields.TargetVersion,
			}
			assert.Equal(t, se.Type(), tt.want)
		})
	}
}

func TestSetProtocolVersionResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgSetProtocolVersionResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &SetProtocolVersionResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, se.Type(), tt.want)
		})
	}
}

func TestSetProtocolVersionResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - invalid field",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SetProtocolVersionResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestGetReaderCapabilities_Type(t *testing.T) {
	type fields struct {
		ReaderCapabilitiesRequestedData ReaderCapability
		Custom                          []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK",
			fields: fields{ReaderCapabilitiesRequestedData: ReaderCapAll, Custom: []Custom{}},
			want:   MsgGetReaderCapabilities,
		},
		{
			name:   "OK - custom",
			fields: fields{ReaderCapabilitiesRequestedData: ReaderCapGeneralDeviceCapabilities, Custom: []Custom{{VendorID: ImpinjSearchMode, Subtype: ImpinjTagReportContentSelector, Data: []byte{'1'}}}},
			want:   MsgGetReaderCapabilities,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetReaderCapabilities{
				ReaderCapabilitiesRequestedData: tt.fields.ReaderCapabilitiesRequestedData,
				Custom:                          tt.fields.Custom,
			}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetReaderCapabilitiesResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus                LLRPStatus
		GeneralDeviceCapabilities *GeneralDeviceCapabilities
		LLRPCapabilities          *LLRPCapabilities
		RegulatoryCapabilities    *RegulatoryCapabilities
		C1G2LLRPCapabilities      *C1G2LLRPCapabilities
		Custom                    []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - nil",
			fields: fields{},
			want:   MsgGetReaderCapabilitiesResponse,
		},
		{
			name: "OK",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, GeneralDeviceCapabilities: &GeneralDeviceCapabilities{
				HasUTCClock:            true,
				DeviceManufacturer:     uint32(PENImpinj),
				Model:                  uint32(SpeedwayR420),
				FirmwareVersion:        "5.14.0.240",
				GPIOCapabilities:       GPIOCapabilities{4, 4},
				MaxSupportedAntennas:   0,
				PerAntennaAirProtocols: nil,
				ReceiveSensitivities:   nil},
				LLRPCapabilities: &LLRPCapabilities{
					CanReportBufferFillWarning:          true,
					SupportsEventsAndReportHolding:      true,
					MaxPriorityLevelSupported:           1,
					MaxROSpecs:                          1,
					MaxSpecsPerROSpec:                   32,
					MaxInventoryParameterSpecsPerAISpec: 1,
					MaxAccessSpecs:                      1508,
					MaxOpSpecsPerAccessSpec:             8,
				},
				C1G2LLRPCapabilities: &C1G2LLRPCapabilities{
					SupportsBlockWrite:       true,
					MaxSelectFiltersPerQuery: 2,
				},
				RegulatoryCapabilities: &RegulatoryCapabilities{
					CountryCode:            840,
					CommunicationsStandard: 1,
					UHFBandCapabilities: &UHFBandCapabilities{
						TransmitPowerLevels: nil,
						FrequencyInformation: FrequencyInformation{
							Hopping: true,
							FrequencyHopTables: []FrequencyHopTable{{
								HopTableID:  1,
								Frequencies: nil,
							}},
						},
						C1G2RFModes: UHFC1G2RFModeTable{
							UHFC1G2RFModeTableEntries: []UHFC1G2RFModeTableEntry{
								{
									ModeID:                0,
									DivideRatio:           DRSixtyFourToThree,
									Modulation:            FM0,
									ForwardLinkModulation: PhaseReversalASK,
									SpectralMask:          SpectralMaskMultiInterrogator,
									BackscatterDataRate:   640000, // actually BLF
									PIERatio:              1500,
									MinTariTime:           6250,
									MaxTariTime:           6250,
								},

								{
									ModeID:                1,
									DivideRatio:           DRSixtyFourToThree,
									Modulation:            Miller2,
									ForwardLinkModulation: PhaseReversalASK,
									SpectralMask:          SpectralMaskMultiInterrogator,
									BackscatterDataRate:   640000, // actually BLF
									PIERatio:              1500,
									MinTariTime:           6250,
									MaxTariTime:           6250,
								},

								{
									ModeID:                2,
									DivideRatio:           DRSixtyFourToThree,
									Modulation:            Miller4,
									ForwardLinkModulation: DoubleSidebandASK,
									SpectralMask:          SpectralMaskDenseInterrogator,
									BackscatterDataRate:   274000, // actually BLF
									PIERatio:              2000,
									MinTariTime:           20000,
									MaxTariTime:           20000,
								},

								{
									ModeID:                3,
									DivideRatio:           DRSixtyFourToThree,
									Modulation:            Miller8,
									ForwardLinkModulation: DoubleSidebandASK,
									SpectralMask:          SpectralMaskDenseInterrogator,
									BackscatterDataRate:   170600, // actually BLF
									PIERatio:              2000,
									MinTariTime:           20000,
									MaxTariTime:           20000,
								},

								{
									ModeID:                4,
									DivideRatio:           DRSixtyFourToThree,
									Modulation:            Miller4,
									ForwardLinkModulation: DoubleSidebandASK,
									SpectralMask:          SpectralMaskMultiInterrogator,
									BackscatterDataRate:   640000, // actually BLF
									PIERatio:              1500,
									MinTariTime:           7140,
									MaxTariTime:           7140,
								},

								// the rest of these are the "auto modes",
								// so the details are non-sense,
								// but they do store the min values in the fields.
								{
									ModeID:              1000,
									BackscatterDataRate: 40000, PIERatio: 1500,
									MinTariTime: 6250, MaxTariTime: 6250,
								},

								{
									ModeID:              1002,
									BackscatterDataRate: 40000, PIERatio: 1500,
									MinTariTime: 6250, MaxTariTime: 6250,
								},

								{
									ModeID:              1003,
									BackscatterDataRate: 40000, PIERatio: 1500,
									MinTariTime: 6250, MaxTariTime: 6250,
								},

								{
									ModeID:              1004,
									BackscatterDataRate: 40000, PIERatio: 1500,
									MinTariTime: 6250, MaxTariTime: 6250,
								},

								{
									ModeID:              1005,
									BackscatterDataRate: 40000, PIERatio: 1500,
									MinTariTime: 6250, MaxTariTime: 6250,
								},
							},
						},
					},
				},
			},
			want: MsgGetReaderCapabilitiesResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetReaderCapabilitiesResponse{
				LLRPStatus:                tt.fields.LLRPStatus,
				GeneralDeviceCapabilities: tt.fields.GeneralDeviceCapabilities,
				LLRPCapabilities:          tt.fields.LLRPCapabilities,
				RegulatoryCapabilities:    tt.fields.RegulatoryCapabilities,
				C1G2LLRPCapabilities:      tt.fields.C1G2LLRPCapabilities,
				Custom:                    tt.fields.Custom,
			}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetReaderCapabilitiesResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus                LLRPStatus
		GeneralDeviceCapabilities *GeneralDeviceCapabilities
		LLRPCapabilities          *LLRPCapabilities
		RegulatoryCapabilities    *RegulatoryCapabilities
		C1G2LLRPCapabilities      *C1G2LLRPCapabilities
		Custom                    []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - nil",
			fields: fields{},
			want:   LLRPStatus{},
		},
		{
			name:   "OK - status success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GetReaderCapabilitiesResponse{
				LLRPStatus:                tt.fields.LLRPStatus,
				GeneralDeviceCapabilities: tt.fields.GeneralDeviceCapabilities,
				LLRPCapabilities:          tt.fields.LLRPCapabilities,
				RegulatoryCapabilities:    tt.fields.RegulatoryCapabilities,
				C1G2LLRPCapabilities:      tt.fields.C1G2LLRPCapabilities,
				Custom:                    tt.fields.Custom,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestAddROSpec_Type(t *testing.T) {
	type fields struct {
		ROSpec ROSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - nil",
			fields: fields{ROSpec: ROSpec{}},
			want:   MsgAddROSpec,
		},
		{
			name:   "OK",
			fields: fields{ROSpec: ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}},
			want:   MsgAddROSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := &AddROSpec{
				ROSpec: tt.fields.ROSpec,
			}
			assert.Equal(t, ad.Type(), tt.want)
		})
	}
}

func TestAddROSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgAddROSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgAddROSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgAddROSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := &AddROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, ad.Type(), tt.want)
		})
	}
}

func TestAddROSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &AddROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestDeleteROSpec_Type(t *testing.T) {
	type fields struct {
		ROSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - ImpinjTagReportContentSelector",
			fields: fields{ROSpecID: ImpinjTagReportContentSelector},
			want:   MsgDeleteROSpec,
		},
		{
			name:   "OK - zero",
			fields: fields{ROSpecID: 0},
			want:   MsgDeleteROSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			de := &DeleteROSpec{
				ROSpecID: tt.fields.ROSpecID,
			}
			assert.Equal(t, de.Type(), tt.want)
		})
	}
}

func TestDeleteROSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgDeleteROSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgDeleteROSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgDeleteROSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			de := &DeleteROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, de.Type(), tt.want)
		})
	}
}

func TestDeleteROSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DeleteROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestStartROSpec_Type(t *testing.T) {
	type fields struct {
		ROSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - ImpinjTagReportContentSelector",
			fields: fields{ROSpecID: ImpinjTagReportContentSelector},
			want:   MsgStartROSpec,
		},
		{
			name:   "OK - zero",
			fields: fields{ROSpecID: 0},
			want:   MsgStartROSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &StartROSpec{
				ROSpecID: tt.fields.ROSpecID,
			}
			assert.Equal(t, st.Type(), tt.want)
		})
	}
}

func TestStartROSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgStartROSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgStartROSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgStartROSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &StartROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, st.Type(), tt.want)
		})
	}
}

func TestStartROSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &StartROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestStopROSpec_Type(t *testing.T) {
	type fields struct {
		ROSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - ImpinjTagReportContentSelector",
			fields: fields{ROSpecID: ImpinjTagReportContentSelector},
			want:   MsgStopROSpec,
		},
		{
			name:   "OK - zero",
			fields: fields{ROSpecID: 0},
			want:   MsgStopROSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &StopROSpec{
				ROSpecID: tt.fields.ROSpecID,
			}
			assert.Equal(t, st.Type(), tt.want)
		})
	}
}

func TestStopROSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgStopROSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgStopROSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgStopROSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &StopROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, st.Type(), tt.want)
		})
	}
}

func TestStopROSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &StopROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestEnableROSpec_Type(t *testing.T) {
	type fields struct {
		ROSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - ImpinjTagReportContentSelector",
			fields: fields{ROSpecID: ImpinjTagReportContentSelector},
			want:   MsgEnableROSpec,
		},
		{
			name:   "OK - zero",
			fields: fields{ROSpecID: 0},
			want:   MsgEnableROSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			en := &EnableROSpec{
				ROSpecID: tt.fields.ROSpecID,
			}
			assert.Equal(t, en.Type(), tt.want)
		})
	}
}

func TestEnableROSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgEnableROSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgEnableROSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgEnableROSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			en := &EnableROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, en.Type(), tt.want)
		})
	}
}

func TestEnableROSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EnableROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestDisableROSpec_Type(t *testing.T) {
	type fields struct {
		ROSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - ImpinjTagReportContentSelector",
			fields: fields{ROSpecID: ImpinjTagReportContentSelector},
			want:   MsgDisableROSpec,
		},
		{
			name:   "OK - zero",
			fields: fields{ROSpecID: 0},
			want:   MsgDisableROSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			di := &DisableROSpec{
				ROSpecID: tt.fields.ROSpecID,
			}
			assert.Equal(t, di.Type(), tt.want)
		})
	}
}

func TestDisableROSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgDisableROSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgDisableROSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgDisableROSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			di := &DisableROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, di.Type(), tt.want)
		})
	}
}

func TestDisableROSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DisableROSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestGetROSpecs_Type(t *testing.T) {
	tests := []struct {
		name string
		want MessageType
	}{
		{
			name: "OK",
			want: MsgGetROSpecs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetROSpecs{}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetROSpecsResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
		ROSpecs    []ROSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - nil",
			fields: fields{},
			want:   MsgGetROSpecsResponse,
		},
		{
			name:   "OK - one ROSpec",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, ROSpecs: []ROSpec{ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}}},
			want:   MsgGetROSpecsResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetROSpecsResponse{
				LLRPStatus: tt.fields.LLRPStatus,
				ROSpecs:    tt.fields.ROSpecs,
			}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetROSpecsResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
		ROSpecs    []ROSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, ROSpecs: []ROSpec{ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}, ROSpecs: []ROSpec{ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}, ROSpecs: []ROSpec{ROSpec{ROSpecID: ImpinjTagReportContentSelector, Priority: 0, ROSpecCurrentState: ROSpecStateActive, ROBoundarySpec: ROBoundarySpec{StartTrigger: ROSpecStartTrigger{GPITrigger: &GPITriggerValue{Port: 0, Event: false, Timeout: 0}}}}}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GetROSpecsResponse{
				LLRPStatus: tt.fields.LLRPStatus,
				ROSpecs:    tt.fields.ROSpecs,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestAddAccessSpec_Type(t *testing.T) {
	type fields struct {
		AccessSpec AccessSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - nil",
			fields: fields{},
			want:   MsgAddAccessSpec,
		},
		{
			name:   "OK",
			fields: fields{AccessSpec: AccessSpec{AccessSpecID: ImpinjTagReportContentSelector, AntennaID: 0, AirProtocolID: AirProtoEPCGlobalClass1Gen2, IsActive: true, ROSpecID: ImpinjPeakRSSI, Trigger: AccessSpecStopTrigger{Trigger: AccessSpecStopTriggerNone, OperationCountValue: 0}, AccessCommand: AccessCommand{}}},
			want:   MsgAddAccessSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := &AddAccessSpec{
				AccessSpec: tt.fields.AccessSpec,
			}
			assert.Equal(t, ad.Type(), tt.want)
		})
	}
}

func TestAddAccessSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgAddAccessSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgAddAccessSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgAddAccessSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ad := &AddAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, ad.Type(), tt.want)
		})
	}
}

func TestAddAccessSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &AddAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestDeleteAccessSpec_Type(t *testing.T) {
	type fields struct {
		AccessSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - zero",
			fields: fields{AccessSpecID: 0},
			want:   MsgDeleteAccessSpec,
		},
		{
			name:   "OK - value",
			fields: fields{AccessSpecID: uint32(5)},
			want:   MsgDeleteAccessSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			de := &DeleteAccessSpec{
				AccessSpecID: tt.fields.AccessSpecID,
			}
			assert.Equal(t, de.Type(), tt.want)
		})
	}
}

func TestDeleteAccessSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgDeleteAccessSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgDeleteAccessSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgDeleteAccessSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			de := &DeleteAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, de.Type(), tt.want)
		})
	}
}

func TestDeleteAccessSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DeleteAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestEnableAccessSpec_Type(t *testing.T) {
	type fields struct {
		AccessSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - zero",
			fields: fields{AccessSpecID: 0},
			want:   MsgEnableAccessSpec,
		},
		{
			name:   "OK - value",
			fields: fields{AccessSpecID: uint32(5)},
			want:   MsgEnableAccessSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			en := &EnableAccessSpec{
				AccessSpecID: tt.fields.AccessSpecID,
			}
			assert.Equal(t, en.Type(), tt.want)
		})
	}
}

func TestEnableAccessSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgEnableAccessSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgEnableAccessSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgEnableAccessSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			en := &EnableAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, en.Type(), tt.want)
		})
	}
}

func TestEnableAccessSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &EnableAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestDisableAccessSpec_Type(t *testing.T) {
	type fields struct {
		AccessSpecID uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - zero",
			fields: fields{AccessSpecID: 0},
			want:   MsgDisableAccessSpec,
		},
		{
			name:   "OK - value",
			fields: fields{AccessSpecID: uint32(5)},
			want:   MsgDisableAccessSpec,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			di := &DisableAccessSpec{
				AccessSpecID: tt.fields.AccessSpecID,
			}
			assert.Equal(t, di.Type(), tt.want)
		})
	}
}

func TestDisableAccessSpecResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgDisableAccessSpecResponse,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgDisableAccessSpecResponse,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgDisableAccessSpecResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			di := &DisableAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, di.Type(), tt.want)
		})
	}
}

func TestDisableAccessSpecResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &DisableAccessSpecResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestGetAccessSpecs_Type(t *testing.T) {
	tests := []struct {
		name string
		want MessageType
	}{
		{
			name: "OK",
			want: MsgGetAccessSpecs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetAccessSpecs{}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetAccessSpecsResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus  LLRPStatus
		AccessSpecs []AccessSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK - nil",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, AccessSpecs: nil},
			want: MsgGetAccessSpecsResponse,
		},
		{
			name: "OK - one AccessSpec",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, AccessSpecs: []AccessSpec{{AccessSpecID: ImpinjTagReportContentSelector, AntennaID: 0, AirProtocolID: AirProtoEPCGlobalClass1Gen2, IsActive: true, ROSpecID: ImpinjPeakRSSI, Trigger: AccessSpecStopTrigger{Trigger: AccessSpecStopTriggerNone, OperationCountValue: 0}, AccessCommand: AccessCommand{}}}},
			want: MsgGetAccessSpecsResponse,
		},

	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetAccessSpecsResponse{
				LLRPStatus:  tt.fields.LLRPStatus,
				AccessSpecs: tt.fields.AccessSpecs,
			}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetAccessSpecsResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus  LLRPStatus
		AccessSpecs []AccessSpec
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name: "OK - nil success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, AccessSpecs: nil},
			want: LLRPStatus{Status: StatusSuccess},
		},
		{
			name: "OK - nil device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}, AccessSpecs: nil},
			want: LLRPStatus{Status: StatusDeviceError},
		},
		{
			name: "OK - one AccessSpec success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, AccessSpecs: []AccessSpec{{AccessSpecID: ImpinjTagReportContentSelector, AntennaID: 0, AirProtocolID: AirProtoEPCGlobalClass1Gen2, IsActive: true, ROSpecID: ImpinjPeakRSSI, Trigger: AccessSpecStopTrigger{Trigger: AccessSpecStopTriggerNone, OperationCountValue: 0}, AccessCommand: AccessCommand{}}}},
			want: LLRPStatus{Status: StatusSuccess},
		},
		{
			name: "OK - one AccessSpec device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}, AccessSpecs: []AccessSpec{{AccessSpecID: ImpinjTagReportContentSelector, AntennaID: 0, AirProtocolID: AirProtoEPCGlobalClass1Gen2, IsActive: true, ROSpecID: ImpinjPeakRSSI, Trigger: AccessSpecStopTrigger{Trigger: AccessSpecStopTriggerNone, OperationCountValue: 0}, AccessCommand: AccessCommand{}}}},
			want: LLRPStatus{Status: StatusDeviceError},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GetAccessSpecsResponse{
				LLRPStatus:  tt.fields.LLRPStatus,
				AccessSpecs: tt.fields.AccessSpecs,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestClientRequestOp_Type(t *testing.T) {
	type fields struct {
		TagReportData TagReportData
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK - defaults",
			fields: fields{TagReportData: TagReportData{
				ROSpecID:                 new(ROSpecID),
				SpecIndex:                new(SpecIndex),
				InventoryParameterSpecID: new(InventoryParameterSpecID),
				AntennaID:                new(AntennaID),
				PeakRSSI:                 new(PeakRSSI),
				ChannelIndex:             new(ChannelIndex),
				FirstSeenUTC:             new(FirstSeenUTC),
				LastSeenUTC:              new(LastSeenUTC),
				TagSeenCount:             new(TagSeenCount),
			}},
			want: MsgClientRequestOp,
		},
		{
			name: "OK",
			fields: fields{TagReportData: TagReportData{
				ROSpecID:                 new(ROSpecID),
				SpecIndex:                new(SpecIndex),
				InventoryParameterSpecID: new(InventoryParameterSpecID),
				AntennaID:                new(AntennaID),
				PeakRSSI:                 new(PeakRSSI),
				ChannelIndex:             new(ChannelIndex),
				FirstSeenUTC:             new(FirstSeenUTC),
				LastSeenUTC:              new(LastSeenUTC),
				TagSeenCount:             new(TagSeenCount),
				EPC96: EPC96{EPC: impinjEnableBool16(2)},
			}},
			want: MsgClientRequestOp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &ClientRequestOp{
				TagReportData: tt.fields.TagReportData,
			}
			assert.Equal(t, cl.Type(), tt.want)
		})
	}
}

func TestClientRequestOpResponse_Type(t *testing.T) {
	type fields struct {
		ClientRequestResponse ClientRequestResponse
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK",
			fields: fields{ClientRequestResponse: ClientRequestResponse{
				AccessSpecID: ImpinjTagReportContentSelector,
				EPCData: EPCData{EPCNumBits: 0, EPC: impinjEnableBool16(5)},
			}},
			want: MsgClientRequestOpResponse,
		},
		{
			name: "OK",
			fields: fields{ClientRequestResponse: ClientRequestResponse{
				AccessSpecID: ImpinjTagReportContentSelector,
				EPCData: EPCData{EPCNumBits: 0, EPC: impinjEnableBool16(5)},
				Custom: &Custom{VendorID: ImpinjSearchMode},
			}},
			want: MsgClientRequestOpResponse,
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &ClientRequestOpResponse{
				ClientRequestResponse: tt.fields.ClientRequestResponse,
			}
			assert.Equal(t, cl.Type(), tt.want)
		})
	}
}

func TestROAccessReport_Type(t *testing.T) {
	type fields struct {
		TagReportData      []TagReportData
		RFSurveyReportData []RFSurveyReportData
		Custom             []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK",
			fields: fields{},
			want: MsgROAccessReport,
		},
		{
			name: "OK",
			fields: fields{TagReportData: []TagReportData{{
				ROSpecID:                 new(ROSpecID),
				SpecIndex:                new(SpecIndex),
				InventoryParameterSpecID: new(InventoryParameterSpecID),
				AntennaID:                new(AntennaID),
				PeakRSSI:                 new(PeakRSSI),
				ChannelIndex:             new(ChannelIndex),
				FirstSeenUTC:             new(FirstSeenUTC),
				LastSeenUTC:              new(LastSeenUTC),
				TagSeenCount:             new(TagSeenCount),
				EPC96: EPC96{EPC: impinjEnableBool16(2)},
			}}, Custom: []Custom{{
				VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'},
			}}},
			want: MsgROAccessReport,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ro := &ROAccessReport{
				TagReportData:      tt.fields.TagReportData,
				RFSurveyReportData: tt.fields.RFSurveyReportData,
				Custom:             tt.fields.Custom,
			}
			assert.Equal(t, ro.Type(), tt.want)
		})
	}
}

func TestKeepAlive_Type(t *testing.T) {
	tests := []struct {
		name string
		want MessageType
	}{
		{
			name: "OK",
			want: MsgKeepAlive,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := &KeepAlive{}
			assert.Equal(t, ke.Type(), tt.want)
		})
	}
}

func TestKeepAliveAck_Type(t *testing.T) {
	tests := []struct {
		name string
		want MessageType
	}{
		{
			name: "OK",
			want: MsgKeepAliveAck,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ke := &KeepAliveAck{}
			assert.Equal(t, ke.Type(), tt.want)
		})
	}
}

func TestReaderEventNotification_Type(t *testing.T) {
	type fields struct {
		ReaderEventNotificationData ReaderEventNotificationData
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK",
			fields: fields{},
			want: MsgReaderEventNotification,
		},
		{
			name: "OK",
			fields: fields{ReaderEventNotificationData: ReaderEventNotificationData{
				Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}},
			}},
			want: MsgReaderEventNotification,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := &ReaderEventNotification{
				ReaderEventNotificationData: tt.fields.ReaderEventNotificationData,
			}
			assert.Equal(t, re.Type(), tt.want)
		})
	}
}

func TestEnableEventsAndReports_Type(t *testing.T) {
	tests := []struct {
		name string
		want MessageType
	}{
		{
			name: "OK",
			want: MsgEnableEventsAndReports,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			en := &EnableEventsAndReports{}
			assert.Equal(t, en.Type(), tt.want)

		})
	}
}

func TestErrorMessage_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   MsgErrorMessage,
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   MsgErrorMessage,
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   MsgErrorMessage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := &ErrorMessage{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, er.Type(), tt.want)
		})
	}
}

func TestErrorMessage_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name:   "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want:   LLRPStatus{Status: StatusSuccess},
		},
		{
			name:   "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want:   LLRPStatus{Status: StatusDeviceError},
		},
		{
			name:   "OK - field invalid",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want:   LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ErrorMessage{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestGetReaderConfig_Type(t *testing.T) {
	type fields struct {
		AntennaID     AntennaID
		RequestedData ReaderConfigRequestedDataType
		GPIPortNum    uint16
		GPOPortNum    uint16
		Custom        []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK",
			fields: fields{Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}},},
			want: MsgGetReaderConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetReaderConfig{
				AntennaID:     tt.fields.AntennaID,
				RequestedData: tt.fields.RequestedData,
				GPIPortNum:    tt.fields.GPIPortNum,
				GPOPortNum:    tt.fields.GPOPortNum,
				Custom:        tt.fields.Custom,
			}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetReaderConfigResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus                  LLRPStatus
		Identification              *Identification
		AntennaProperties           []AntennaProperties
		AntennaConfigurations       []AntennaConfiguration
		ReaderEventNotificationSpec *ReaderEventNotificationSpec
		ROReportSpec                *ROReportSpec
		AccessReportSpec            *AccessReportSpec
		LLRPConfigurationStateValue *LLRPConfigurationStateValue
		KeepAliveSpec               *KeepAliveSpec
		GPIPortCurrentStates        []GPIPortCurrentState
		GPOWriteData                []GPOWriteData
		EventsAndReports            *EventsAndReports
		Custom                      []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK",
			fields: fields{Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}},},
			want: MsgGetReaderConfigResponse,
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ge := &GetReaderConfigResponse{
				LLRPStatus:                  tt.fields.LLRPStatus,
				Identification:              tt.fields.Identification,
				AntennaProperties:           tt.fields.AntennaProperties,
				AntennaConfigurations:       tt.fields.AntennaConfigurations,
				ReaderEventNotificationSpec: tt.fields.ReaderEventNotificationSpec,
				ROReportSpec:                tt.fields.ROReportSpec,
				AccessReportSpec:            tt.fields.AccessReportSpec,
				LLRPConfigurationStateValue: tt.fields.LLRPConfigurationStateValue,
				KeepAliveSpec:               tt.fields.KeepAliveSpec,
				GPIPortCurrentStates:        tt.fields.GPIPortCurrentStates,
				GPOWriteData:                tt.fields.GPOWriteData,
				EventsAndReports:            tt.fields.EventsAndReports,
				Custom:                      tt.fields.Custom,
			}
			assert.Equal(t, ge.Type(), tt.want)
		})
	}
}

func TestGetReaderConfigResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus                  LLRPStatus
		Identification              *Identification
		AntennaProperties           []AntennaProperties
		AntennaConfigurations       []AntennaConfiguration
		ReaderEventNotificationSpec *ReaderEventNotificationSpec
		ROReportSpec                *ROReportSpec
		AccessReportSpec            *AccessReportSpec
		LLRPConfigurationStateValue *LLRPConfigurationStateValue
		KeepAliveSpec               *KeepAliveSpec
		GPIPortCurrentStates        []GPIPortCurrentState
		GPOWriteData                []GPOWriteData
		EventsAndReports            *EventsAndReports
		Custom                      []Custom
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name: "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}, Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}}},
			want: LLRPStatus{Status: StatusSuccess},
		},
		{
			name: "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}, Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}}},
			want: LLRPStatus{Status: StatusDeviceError},
		},
		{
			name: "OK - invalid field",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}, Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}}},
			want: LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &GetReaderConfigResponse{
				LLRPStatus:                  tt.fields.LLRPStatus,
				Identification:              tt.fields.Identification,
				AntennaProperties:           tt.fields.AntennaProperties,
				AntennaConfigurations:       tt.fields.AntennaConfigurations,
				ReaderEventNotificationSpec: tt.fields.ReaderEventNotificationSpec,
				ROReportSpec:                tt.fields.ROReportSpec,
				AccessReportSpec:            tt.fields.AccessReportSpec,
				LLRPConfigurationStateValue: tt.fields.LLRPConfigurationStateValue,
				KeepAliveSpec:               tt.fields.KeepAliveSpec,
				GPIPortCurrentStates:        tt.fields.GPIPortCurrentStates,
				GPOWriteData:                tt.fields.GPOWriteData,
				EventsAndReports:            tt.fields.EventsAndReports,
				Custom:                      tt.fields.Custom,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestSetReaderConfig_Type(t *testing.T) {
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
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK",
			fields: fields{},
			want: MsgSetReaderConfig,
		},
		{
			name: "OK",
			fields: fields{Custom: []Custom{{VendorID: 0, Subtype: ImpinjTagReportContentSelector, Data: []byte{'b'}}},},
			want: MsgSetReaderConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &SetReaderConfig{
				ResetToFactoryDefaults:      tt.fields.ResetToFactoryDefaults,
				ReaderEventNotificationSpec: tt.fields.ReaderEventNotificationSpec,
				AntennaProperties:           tt.fields.AntennaProperties,
				AntennaConfigurations:       tt.fields.AntennaConfigurations,
				ROReportSpec:                tt.fields.ROReportSpec,
				AccessReportSpec:            tt.fields.AccessReportSpec,
				KeepAliveSpec:               tt.fields.KeepAliveSpec,
				GPOWriteData:                tt.fields.GPOWriteData,
				GPIPortCurrentStates:        tt.fields.GPIPortCurrentStates,
				EventsAndReports:            tt.fields.EventsAndReports,
				Custom:                      tt.fields.Custom,
			}
			assert.Equal(t, se.Type(), tt.want)
		})
	}
}

func TestSetReaderConfigResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want: MsgSetReaderConfigResponse,
		},
		{
			name: "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want: MsgSetReaderConfigResponse,
		},
		{
			name: "OK - invalid field",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want: MsgSetReaderConfigResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &SetReaderConfigResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, se.Type(), tt.want)
		})
	}
}

func TestSetReaderConfigResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name: "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want: LLRPStatus{Status: StatusSuccess},
		},
		{
			name: "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want: LLRPStatus{Status: StatusDeviceError},
		},
		{
			name: "OK - invalid field",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want: LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &SetReaderConfigResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestCloseConnection_Type(t *testing.T) {
	tests := []struct {
		name string
		want MessageType
	}{
		{
			name: "OK",
			want: MsgCloseConnection,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &CloseConnection{}
			assert.Equal(t, cl.Type(), tt.want)
		})
	}
}

func TestCloseConnectionResponse_Type(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want: MsgCloseConnectionResponse,
		},
		{
			name: "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want: MsgCloseConnectionResponse,
		},
		{
			name: "OK - invalid field",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want: MsgCloseConnectionResponse,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := &CloseConnectionResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, cl.Type(), tt.want)
		})
	}
}

func TestCloseConnectionResponse_Status(t *testing.T) {
	type fields struct {
		LLRPStatus LLRPStatus
	}
	tests := []struct {
		name   string
		fields fields
		want   LLRPStatus
	}{
		{
			name: "OK - success",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusSuccess}},
			want: LLRPStatus{Status: StatusSuccess},
		},
		{
			name: "OK - device err",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusDeviceError}},
			want: LLRPStatus{Status: StatusDeviceError},
		},
		{
			name: "OK - invalid field",
			fields: fields{LLRPStatus: LLRPStatus{Status: StatusFieldInvalid}},
			want: LLRPStatus{Status: StatusFieldInvalid},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &CloseConnectionResponse{
				LLRPStatus: tt.fields.LLRPStatus,
			}
			assert.Equal(t, m.Status(), tt.want)
		})
	}
}

func TestCustomMessage_Type(t *testing.T) {
	type fields struct {
		VendorID       uint32
		MessageSubtype uint8
		Data           []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   MessageType
	}{
		{
			name: "OK - ImpinjTagReportContentSelector",
			fields: fields{VendorID: ImpinjTagReportContentSelector, MessageSubtype: 0, Data: []byte{'b'}},
			want: MsgCustomMessage,
		},
		{
			name: "OK - ImpinjPeakRSSI",
			fields: fields{VendorID: ImpinjPeakRSSI, MessageSubtype: 10, Data: []byte{'c'}},
			want: MsgCustomMessage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cu := &CustomMessage{
				VendorID:       tt.fields.VendorID,
				MessageSubtype: tt.fields.MessageSubtype,
				Data:           tt.fields.Data,
			}
			assert.Equal(t, cu.Type(), tt.want)
		})
	}
}
