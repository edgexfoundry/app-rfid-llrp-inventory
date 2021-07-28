//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// rssiPtr converts a float64 to a PeakRSSI pointer
func rssiPtr(val float64) *PeakRSSI {
	peak := PeakRSSI(val)
	return &peak
}

// int16ToBytes converts a 16-bit int to a 2-byte []byte
func int16ToBytes(i int16) []byte {
	return []byte{byte(i >> 8), byte(i)}
}

// makeCustomRSSI creates an ImpinjPeakRSSI Custom struct with the specified data bytes
func makeCustomRSSI(data []byte) Custom {
	return Custom{
		VendorID: uint32(PENImpinj),
		Subtype:  ImpinjPeakRSSI,
		Data:     data,
	}
}

func TestExtractRSSI(t *testing.T) {
	tests := []struct {
		name     string
		data     TagReportData
		expected float64
		hasRSSI  bool
	}{
		{
			name:     "default value",
			data:     TagReportData{PeakRSSI: new(PeakRSSI)},
			expected: float64(0),
			hasRSSI:  true,
		},
		{
			name:     "peak value only",
			data:     TagReportData{PeakRSSI: rssiPtr(-35)},
			expected: float64(-35),
			hasRSSI:  true,
		},
		{
			name:    "nil",
			data:    TagReportData{PeakRSSI: nil},
			hasRSSI: false,
		},
		{
			name:     "custom only",
			data:     TagReportData{Custom: []Custom{makeCustomRSSI(int16ToBytes(-4750))}},
			expected: -47.5,
			hasRSSI:  true,
		},
		{
			name:    "custom nil, missing peak rssi",
			data:    TagReportData{Custom: []Custom{makeCustomRSSI(nil)}},
			hasRSSI: false,
		},
		{
			name:    "custom and peak rssi both nil",
			data:    TagReportData{PeakRSSI: nil, Custom: []Custom{makeCustomRSSI(nil)}},
			hasRSSI: false,
		},
		{
			name:     "custom nil, fallback to peak rssi",
			data:     TagReportData{PeakRSSI: rssiPtr(-50), Custom: []Custom{makeCustomRSSI(nil)}},
			expected: float64(-50),
			hasRSSI:  true,
		},
		{
			name:     "prefer custom over peak rssi",
			data:     TagReportData{PeakRSSI: rssiPtr(-62), Custom: []Custom{makeCustomRSSI(int16ToBytes(-6150))}},
			expected: -61.5,
			hasRSSI:  true,
		},
		{
			name:    "custom - wrong data length",
			data:    TagReportData{Custom: []Custom{makeCustomRSSI([]byte{1, 2, 3, 4})}},
			hasRSSI: false,
		},
		{
			name: "custom - wrong subtype",
			data: TagReportData{Custom: []Custom{{
				VendorID: uint32(PENImpinj),
				Subtype:  ImpinjEnablePeakRSSI, // note: wrong subtype
				Data:     int16ToBytes(-6850),
			}}},
			hasRSSI: false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			rssi, hasRSSI := test.data.ExtractRSSI()
			assert.Equal(t, test.hasRSSI, hasRSSI)
			if test.hasRSSI { // only check expected rssi if we actually expect an rssi
				assert.Equal(t, test.expected, rssi)
			}
		})
	}
}

func TestWordsToHex(t *testing.T) {
	var tests = []struct {
		name  string
		words []uint16
		want  string
	}{
		{
			name:  "OK - empty",
			words: []uint16{},
			want:  "",
		},
		{
			name:  "OK - range: 0-7",
			words: []uint16{0, 1, 2, 3, 4, 5, 6, 7},
			want:  "00000001000200030004000500060007",
		},
		{
			name:  "OK - range: 8-15",
			words: []uint16{8, 9, 10, 11, 12, 13, 14, 15},
			want:  "00080009000a000b000c000d000e000f",
		},
		{
			name:  "OK - range: f0-f7",
			words: []uint16{0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7},
			want:  "00f000f100f200f300f400f500f600f7",
		},
		{
			name:  "OK - range: f8-ff",
			words: []uint16{0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff},
			want:  "00f800f900fa00fb00fc00fd00fe00ff",
		},
		{
			name:  "OK - 0x3fad",
			words: []uint16{0x3fad},
			want:  "3fad",
		},
		{
			name:  "OK - letter g",
			words: []uint16{uint16('g')},
			want:  "0067",
		},
		{
			name:  "OK - 0xfa32 0x14ae",
			words: []uint16{0xfa32, 0x14ae},
			want:  "fa3214ae",
		},
		{
			name:  "OK - range: e3, a1",
			words: []uint16{0xe3, 0xa1},
			want:  "00e300a1",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			res := wordsToHex(test.words)
			assert.Equal(t, test.want, res)
		})
	}

	// Re-use the input data for testing ReadDataAsHex as well
	for _, test := range tests {
		test := test
		t.Run("ReadDataAsHex_"+test.name, func(t *testing.T) {
			report := TagReportData{
				C1G2ReadOpSpecResult: &C1G2ReadOpSpecResult{
					C1G2ReadOpSpecResultType: 0,
					OpSpecID:                 0,
					Data:                     test.words,
				},
			}
			actual, ok := report.ReadDataAsHex()
			assert.Equal(t, actual, test.want)
			assert.True(t, ok)
		})
	}
}

func TestReadDataAsHex(t *testing.T) {
	// Note: See also `TestWordsToHex` for more tests of `ReadDataAsHex`
	tests := []struct {
		name     string
		report   TagReportData
		expected string
		wantOk   bool
	}{
		{
			name:     "nil",
			report:   TagReportData{C1G2ReadOpSpecResult: nil},
			expected: "",
			wantOk:   false,
		},
		{
			name:     "default values",
			report:   TagReportData{C1G2ReadOpSpecResult: new(C1G2ReadOpSpecResult)},
			expected: wordsToHex([]uint16{}),
			wantOk:   true,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			actual, ok := test.report.ReadDataAsHex()
			require.Equal(t, test.wantOk, ok)
			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestCustomIs(t *testing.T) {
	tests := []struct {
		name    string
		custom  Custom
		vendor  VendorPEN
		subtype CustomParamSubtype
		success bool
	}{
		{
			name: "OK",
			custom: Custom{
				VendorID: uint32(PENImpinj),
				Subtype:  ImpinjTagReportContentSelector,
				Data:     impinjEnableBool16(ImpinjEnablePeakRSSI),
			},
			vendor:  PENImpinj,
			subtype: ImpinjTagReportContentSelector,
			success: true,
		},
		{
			name: "mismatched subtype",
			custom: Custom{
				VendorID: uint32(PENImpinj),
				Subtype:  ImpinjTagReportContentSelector,
				Data:     impinjEnableBool16(ImpinjEnablePeakRSSI),
			},
			vendor:  PENImpinj,
			subtype: ImpinjSearchMode,
			success: false,
		},
		{
			name: "mismatched vendor",
			custom: Custom{
				VendorID: uint32(PENImpinj),
				Subtype:  ImpinjTagReportContentSelector,
				Data:     impinjEnableBool16(ImpinjEnablePeakRSSI),
			},
			vendor:  PENAlien,
			subtype: ImpinjTagReportContentSelector,
			success: false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			success := test.custom.Is(test.vendor, test.subtype)
			assert.Equal(t, success, test.success)
		})
	}
}
