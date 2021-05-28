//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math"
	"reflect"
	"testing"
	"testing/quick"
)

// testROSpecProperties is a helper function
// that validates various properties on an ROSpec
// that should always be true (of an ROSpec to be sent to a Reader),
// regardless of how that ROSpec is generated.
func testROSpecProperties(t *testing.T, spec *ROSpec) {
	t.Helper()

	// These assume we're creating a new spec
	assert.NotZero(t, spec.ROSpecID)
	assert.Equal(t, spec.ROSpecCurrentState, ROSpecStateDisabled)

	// We're currently controlling reporting at the ReaderConfig level,
	// so we don't want to override it in the ROSpec itself.
	assert.Nil(t, spec.ROReportSpec)
	assert.NotZero(t, len(spec.AISpecs))
}

func TestImpinjEnableBool16(t *testing.T) {
	require.NoError(t, quick.Check(func(subtype uint32) bool {
		data := impinjEnableBool16(subtype)
		if len(data) != int(binary.BigEndian.Uint16(data[2:])) {
			return false
		}

		if binary.BigEndian.Uint16(data) != 1023 {
			return false
		}

		if binary.BigEndian.Uint32(data[4:]) != uint32(PENImpinj) {
			return false
		}

		if binary.BigEndian.Uint32(data[8:]) != subtype {
			return false
		}

		if binary.BigEndian.Uint16(data[12:]) != 1 {
			return false
		}

		return true
	}, nil))
}

var fccFreqs = []Kilohertz{
	902750, 903250, 903750, 904250, 904750, 905250, 905750,
	906250, 906750, 907250, 907750, 908250, 908750, 909250,
	909750, 910250, 910750, 911250, 911750, 912250, 912750,
	913250, 913750, 914250, 914750, 915250, 915750, 916250,
	916750, 917250, 917750, 918250, 918750, 919250, 919750,
	920250, 920750, 921250, 921750, 922250, 922750, 923250,
	923750, 924250, 924750, 925250, 925750, 926250, 926750,
	927250,
}

// newImpinjCaps returns capabilities matching an Impinj Reader in an FCC region.
func newImpinjCaps(t *testing.T) *GetReaderCapabilitiesResponse {
	t.Helper()
	powerTable := make([]TransmitPowerLevelTableEntry, 81)
	for i := range powerTable {
		powerTable[i] = TransmitPowerLevelTableEntry{
			Index:              uint16(i + 1),
			TransmitPowerValue: MillibelMilliwatt(i*25 + 1000),
		}
	}

	receiveTable := make([]ReceiveSensitivityTableEntry, 42)
	for i := range receiveTable {
		receiveTable[i] = ReceiveSensitivityTableEntry{
			Index:              uint16(i + 1),
			ReceiveSensitivity: uint16(i + 9),
		}
	}
	receiveTable[0].ReceiveSensitivity = 0

	assert.Equal(t, len(fccFreqs), 50)

	// Build the frequency list by appending entries from the FCC list,
	// where we step the index into the FCC list by some generator of Z/Z(50)
	// I've chosen 7, but other good options are 3, 9, 11, 13, 17, 21, etc.
	// The important thing is just that it doesn't have a factor of 2 or 5.
	hopTableFreqs := make([]Kilohertz, 50)
	for i, j := 0, 0; i < 50; i, j = i+1, (j+7)%50 {
		hopTableFreqs[i] = fccFreqs[j]
	}

	const numAntennas = 4
	airProto := make([]PerAntennaAirProtocol, numAntennas)
	for i := range airProto {
		airProto[i] = PerAntennaAirProtocol{
			AntennaID:      AntennaID(i + 1),
			AirProtocolIDs: []AirProtocolIDType{AirProtoEPCGlobalClass1Gen2},
		}
	}

	return &GetReaderCapabilitiesResponse{
		GeneralDeviceCapabilities: &GeneralDeviceCapabilities{
			HasUTCClock:            true,
			DeviceManufacturer:     uint32(PENImpinj),
			Model:                  uint32(SpeedwayR420),
			FirmwareVersion:        "5.14.0.240",
			GPIOCapabilities:       GPIOCapabilities{4, 4},
			MaxSupportedAntennas:   numAntennas,
			PerAntennaAirProtocols: airProto,
			ReceiveSensitivities:   receiveTable,
		},

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
				TransmitPowerLevels: powerTable,
				FrequencyInformation: FrequencyInformation{
					Hopping: true,
					FrequencyHopTables: []FrequencyHopTable{{
						HopTableID:  1,
						Frequencies: hopTableFreqs,
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
	}
}

func TestImpinjDevice_invalid(t *testing.T) {
	caps := newImpinjCaps(t)
	caps.RegulatoryCapabilities.UHFBandCapabilities.C1G2RFModes.UHFC1G2RFModeTableEntries = nil
	_, err := NewImpinjDevice(caps)
	require.Error(t, err)
}

func TestImpinjDevice_NewConfig(t *testing.T) {
	caps := newImpinjCaps(t)
	d, err := NewImpinjDevice(caps)
	require.NoError(t, err)
	assert.False(t, d.stateAware)
	assert.True(t, d.allowsHop)
	assert.Equal(t, d.nSpecsPerRO, caps.LLRPCapabilities.MaxSpecsPerROSpec)

	nFreqs := len(caps.RegulatoryCapabilities.UHFBandCapabilities.
		FrequencyInformation.FrequencyHopTables[0].Frequencies)
	assert.Equal(t, int(d.nFreqs), nFreqs)
	assert.Equal(t, d.nGPIs, caps.GeneralDeviceCapabilities.GPIOCapabilities.NumGPIs)

	// make sure we removed the meaningless Autoset modes
	assert.Equal(t, len(d.modes), 5)

	for i := range d.pwrMinToMax[1:] {
		assert.Less(t, d.pwrMinToMax[i].TransmitPowerValue, d.pwrMinToMax[i+1].TransmitPowerValue)

		// The next few tests assume power values are more than 0.01 dBm apart.
		// They
		pwrAtI := d.pwrMinToMax[i]

		// If we look for a power entry that exists, we should get that exact value.
		pIdx, pValue := d.findPower(pwrAtI.TransmitPowerValue)
		assert.NotZero(t, pIdx)
		assert.Equal(t, pwrAtI.TransmitPowerValue, pValue)
		assert.Equal(t, pwrAtI.Index, pIdx)

		// If we search for a power just above this one,
		// we should get back the same power value,
		// assuming power values are more than 0.01 dBm apart.
		pIdx, pValue = d.findPower(pwrAtI.TransmitPowerValue + 1)
		assert.Equal(t, pwrAtI.Index, pIdx)
		pIdx, pValue = d.findPower(pwrAtI.TransmitPowerValue + 1)
		assert.NotZero(t, pIdx)

		// If we search for a power just below this one,
		// we should get back a power value less the target,
		assert.Greater(t, pwrAtI.TransmitPowerValue+1, pValue)
	}

	// Look for a power lower than the lowest power.
	// It should yield the lowest power value.
	lowest := d.pwrMinToMax[0]
	pIdx, pValue := d.findPower(lowest.TransmitPowerValue - 1)
	assert.Equal(t, lowest.Index, pIdx)
	assert.Equal(t, lowest.TransmitPowerValue, pValue)
	assert.NotNil(t, d.NewConfig())
}

func TestBasicDevice_NewROSpec(t *testing.T) {
	caps := newImpinjCaps(t)
	d, err := NewBasicDevice(caps)
	require.NoError(t, err)

	for _, b := range []Behavior{
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}},
		{ScanType: ScanNormal, Power: PowerTarget{Max: 3000}},
		{ScanType: ScanDeep, Power: PowerTarget{Max: 3000}},
		{ScanType: ScanFast, Power: PowerTarget{Max: math.MaxInt16}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, Duration: math.MaxUint16},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 1}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 2}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 3}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 4}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, ImpinjOptions: &ImpinjOptions{SuppressMonza: true}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, ImpinjOptions: &ImpinjOptions{SuppressMonza: false}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, Frequencies: []Kilohertz{fccFreqs[0], fccFreqs[1], fccFreqs[2]}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, Frequencies: []Kilohertz{}},
	} {
		r, err := d.NewROSpec(b, Environment{})
		assert.NoError(t, err)
		assert.NotNil(t, r)
		testROSpecProperties(t, r)
	}

	for _, b := range []Behavior{
		{ScanType: ScanFast, Power: PowerTarget{Max: 30}},
		{ScanType: 5, Power: PowerTarget{Max: math.MinInt16}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 0}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 5}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: math.MaxUint16}},
	} {
		_, err := d.NewROSpec(b, Environment{})
		assert.Error(t, err)
	}
}

func TestBasicDevice_NewROSpec_noHopThisTime(t *testing.T) {
	caps := newImpinjCaps(t)
	freqInfo := &caps.RegulatoryCapabilities.UHFBandCapabilities.FrequencyInformation
	freqInfo.Hopping = false

	_, err := NewBasicDevice(caps)
	require.Error(t, err)
	freqInfo.FrequencyHopTables = nil
	freqInfo.FixedFrequencyTable = &FixedFrequencyTable{Frequencies: fccFreqs}
	d, err := NewBasicDevice(caps)
	assert.NoError(t, err)

	for _, b := range []Behavior{
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, Frequencies: []Kilohertz{fccFreqs[2]}},
	} {
		r, err := d.NewROSpec(b, Environment{})
		assert.NoError(t, err)
		assert.NotNil(t, r)

		for i, ai := range r.AISpecs {
			for j, ips := range ai.InventoryParameterSpecs {
				for k, ac := range ips.AntennaConfigurations {
					assert.NotNil(t, ac.RFTransmitter)

					// 3 because index is 1-based, and above we used fccFreq[2]
					assert.Equal(t, ac.RFTransmitter.ChannelIndex, uint16(3), fmt.Sprintf("invalid channel index for (%d, %d, %d): %d", i, j, k, ac.RFTransmitter.ChannelIndex))
				}
			}
		}
	}

	for _, b := range []Behavior{
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, Frequencies: []Kilohertz{}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}},
		{ScanType: ScanNormal, Power: PowerTarget{Max: 3000}},
		{ScanType: ScanDeep, Power: PowerTarget{Max: 3000}},
		{ScanType: ScanFast, Power: PowerTarget{Max: math.MaxInt16}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, Duration: math.MaxUint16},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 1}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 2}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 3}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 4}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, ImpinjOptions: &ImpinjOptions{SuppressMonza: true}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, ImpinjOptions: &ImpinjOptions{SuppressMonza: false}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 30}},
		{ScanType: 5, Power: PowerTarget{Max: math.MinInt16}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 0}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: 5}},
		{ScanType: ScanFast, Power: PowerTarget{Max: 3000}, GPITrigger: &GPITrigger{Port: math.MaxUint16}},
	} {
		r, err := d.NewROSpec(b, Environment{})
		assert.Error(t, err, "expected an error for behavior %+v: %+v", b, r)
	}
}

func TestFastestAt(t *testing.T) {
	caps := newImpinjCaps(t)
	d, err := NewImpinjDevice(caps)
	require.NoError(t, err)

	// find best should return something, regardless of the number of readers
	for i := 0; i < 100; i++ {
		best, m := d.findBestMode(0)
		assert.Less(t, best, len(d.modes)-1, fmt.Sprintf("find best returned an invalid mode: %d, %+v", best, m))
	}

	type expMode struct {
		mask           SpectralMaskType
		modeIdx        int
		shouldHaveMode bool
	}

	// because there's a Dense mask, there's always some mode available
	for _, testCase := range []expMode{
		{SpectralMaskUnknown, 0, true},
		{SpectralMaskSingleInterrogator, 0, true},
		{SpectralMaskMultiInterrogator, 0, true},
		{SpectralMaskDenseInterrogator, 2, true},
	} {
		bestIdx, ok := d.fastestAt(testCase.mask)
		if !testCase.shouldHaveMode {
			assert.False(t, ok)
			continue
		}
		assert.True(t, ok)
		assert.Equal(t, testCase.modeIdx, bestIdx)
	}

	// if we remove the denser masks, we shouldn't get modes at those higher levels
	for i := range d.modes {
		d.modes[i].SpectralMask = SpectralMaskSingleInterrogator
	}

	for _, testCase := range []expMode{
		{SpectralMaskUnknown, 0, true},
		{SpectralMaskSingleInterrogator, 0, true},
		{SpectralMaskMultiInterrogator, 0, false},
		{SpectralMaskDenseInterrogator, 0, false},
	} {
		bestIdx, ok := d.fastestAt(testCase.mask)
		if !testCase.shouldHaveMode {
			assert.False(t, ok)
			continue
		}
		assert.True(t, ok)
		assert.Equal(t, testCase.modeIdx, bestIdx)
	}

	// find best should return something, regardless of the number of readers
	for i := 0; i < 100; i++ {
		best, m := d.findBestMode(0)
		assert.Less(t, best, len(d.modes)-1, fmt.Sprintf("find best returned an invalid mode: %d, %+v", best, m))
	}
}

func TestBehavior_Boundary_zero(t *testing.T) {
	b := Behavior{}

	bound := b.Boundary()
	checkStartImmediate(t, bound)
	checkStopNone(t, bound)

	b.Duration = 10
	bound = b.Boundary()
	checkStartNone(t, bound)
	checkStopDuration(t, bound)

	b.Duration = 0
	b.GPITrigger = &GPITrigger{}
	bound = b.Boundary()
	checkStartGPI(t, bound)
	checkStopNone(t, bound)
}

func checkStartImmediate(t *testing.T, spec ROBoundarySpec) {
	t.Helper()
	require.Equal(t, spec.StartTrigger.Trigger, ROStartTriggerImmediate)
	require.Nil(t, spec.StartTrigger.GPITrigger)
	require.Nil(t, spec.StartTrigger.PeriodicTrigger)
}

func checkStartNone(t *testing.T, spec ROBoundarySpec) {
	t.Helper()
	require.Equal(t, spec.StartTrigger.Trigger, ROStartTriggerNone)
	require.Nil(t, spec.StartTrigger.GPITrigger)
	require.Nil(t, spec.StartTrigger.PeriodicTrigger)
}

func checkStartGPI(t *testing.T, spec ROBoundarySpec) {
	t.Helper()
	require.Equal(t, spec.StartTrigger.Trigger, ROStartTriggerGPI)
	require.NotNil(t, spec.StartTrigger.GPITrigger)
	require.Nil(t, spec.StartTrigger.PeriodicTrigger)
}

func checkStopNone(t *testing.T, spec ROBoundarySpec) {
	t.Helper()
	require.Equal(t, spec.StopTrigger.Trigger, ROStopTriggerNone)
	require.Zero(t, spec.StopTrigger.DurationTriggerValue)
	require.Nil(t, spec.StopTrigger.GPITriggerValue)
}

func checkStopDuration(t *testing.T, spec ROBoundarySpec) {
	t.Helper()
	require.Equal(t, spec.StopTrigger.Trigger, ROStopTriggerDuration)
	require.NotZero(t, spec.StopTrigger.DurationTriggerValue)
	require.Nil(t, spec.StopTrigger.GPITriggerValue)
}

func TestMarshalBehaviorText(t *testing.T) {
	// These tests are really just a sanity check
	// to validate assumptions about json marshaling.
	// They just marshal the interface v to JSON
	// and verify the data matches,
	// then unmarshal that back to a new pointer
	// with the same type as v,
	// and validates it matches the original value.

	tests := []struct {
		name       string
		val        interface{}
		data       []byte
		shouldFail bool
	}{
		{"fast", ScanFast, []byte(`"Fast"`), false},
		{"normal", ScanNormal, []byte(`"Normal"`), false},
		{"deep", ScanDeep, []byte(`"Deep"`), false},
		{"unknownScan", ScanType(501), nil, true},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(testCase.val)
			if testCase.shouldFail {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, got, testCase.data)

			newInst := reflect.New(reflect.TypeOf(testCase.val))
			ptr := newInst.Interface()
			err = json.Unmarshal(testCase.data, ptr)
			assert.NoError(t, err)

			newVal := newInst.Elem().Interface()
			assert.Equal(t, newVal, testCase.val)
		})
	}
}
