//
// Copyright (C) 2021 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import "encoding/binary"

const hexChars = "0123456789abcdef"

// wordsToHex converts an array of 16-bit words to a hex string.
//
// This is essentially the same method as hex.EncodeToString,
// but operates on []uint16 instead of []byte.
func wordsToHex(src []uint16) string {
	dst := make([]byte, len(src)*4)

	i := 0
	for _, word := range src {
		dst[i+0] = hexChars[(word>>0xC)&0xF]
		dst[i+1] = hexChars[(word>>0x8)&0xF]
		dst[i+2] = hexChars[(word>>0x4)&0xF]
		dst[i+3] = hexChars[(word>>0x0)&0xF]
		i += 4
	}

	return string(dst)
}

// ExtractRSSI returns the RSSI value from TagReportData, if present.
//
// If the report includes a Custom Impinj RSSI parameter, it returns that.
// Because those values are dBm x100, it converts it to dBm (by dividing by 100),
// and hence the returned value is a floats instead of an int.
func (rt *TagReportData) ExtractRSSI() (float64, bool) {
	for _, c := range rt.Custom {
		if c.Is(PENImpinj, ImpinjPeakRSSI) && len(c.Data) == 2 {
			// #nosec G115
			return float64(int16(binary.BigEndian.Uint16(c.Data))) / 100.0, true // dBm x100
		}
	}

	if rt.PeakRSSI != nil {
		return float64(*rt.PeakRSSI), true
	}
	return 0, false
}

// ReadDataAsHex returns a hex string representation of a ReadOpSpecResult
// if the TagReportData has one and its result type indicates success.
func (rt *TagReportData) ReadDataAsHex() (data string, ok bool) {
	if rt.C1G2ReadOpSpecResult == nil {
		return
	}

	res := rt.C1G2ReadOpSpecResult
	if res.C1G2ReadOpSpecResultType == 0 {
		data = wordsToHex(res.Data)
		ok = true
	}

	return
}

// Is returns true if the Custom receiver is the specified Vendor and Subtype.
func (c *Custom) Is(vendor VendorPEN, subtype CustomParamSubtype) bool {
	return VendorPEN(c.VendorID) == vendor && c.Subtype == subtype
}
