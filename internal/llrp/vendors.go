//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

//go:generate stringer -type=VendorPEN,ImpinjModel -output vendors_string.go

package llrp

import "encoding/binary"

type VendorPEN uint32

const (
	PENImpinj = VendorPEN(25882)
	PENAlien  = VendorPEN(17996)
	PENZebra  = VendorPEN(10642)
)

type ImpinjModel uint32

const (
	SpeedwayR220 = ImpinjModel(2001001)
	SpeedwayR420 = ImpinjModel(2001002)
	XPortal      = ImpinjModel(2001003)
	XArrayWM     = ImpinjModel(2001004)
	XArrayEAP    = ImpinjModel(2001006)
	XArray       = ImpinjModel(2001007)
	XSpan        = ImpinjModel(2001008)
	SpeedwayR120 = ImpinjModel(2001009)
	R700         = ImpinjModel(2001052)
)

func (c *Custom) Is(idType VendorPEN, subtype uint32) bool {
	return VendorPEN(c.VendorID) == idType && c.Subtype == subtype
}

type ImpinjParamSubtype = uint32

// impinjSearchMode is like a really limited version of standard state-aware filtering
// with added ambiguity about what C1G2 commands the Reader might send.
type impinjSearchMode = uint16

const (
	ImpinjEnablePeakRSSI           = ImpinjParamSubtype(53)
	ImpinjPeakRSSI                 = ImpinjParamSubtype(57)
	ImpinjTagReportContentSelector = ImpinjParamSubtype(50)
	ImpinjSearchMode               = ImpinjParamSubtype(23)

	// impSearchReaderSelected is the "default" search mode.
	// There's no way to know exactly what it will do.
	impSearchReaderSelected = impinjSearchMode(0)

	// impjSingleTarget is Impinj's "Single Target".
	// It's just a normal Query with the Target field set to A.
	impjSingleTarget = impinjSearchMode(1)

	// impjSingleTargetReset is Impinj's "Single Target Reset".
	// It's just a normal Query with the Target field set to B.
	impjSingleTargetReset = impinjSearchMode(5)

	// impjSingleTargetSuppressed is Impinj's
	// "Single Target Inventory with Suppression (aka TagFocus)".
	// This is identical to "Single Target",
	// but instructs Impinj Monza tags to refresh their S1 flag persistence:
	// https://platform.impinj.com/indy/itk/latest/Glossary/Glossary.html#term-tagfocus
	//
	// Importantly, it only makes sense if used with Session 1,
	// and only if you've got mostly Impinj Monza tags.
	// Otherwise it's exactly the same as querying A->B, but probably slower.
	impjSingleTargetSuppressed = impinjSearchMode(3)

	// impjDualTarget is Impinj's "Dual Target Inventory",
	// which queries A->B til quiet, then B->A til quiet,
	// something you'd normally do with two AISpecs and State Aware singulation.
	//
	// In S0, you're likely to never move from step 1 to step 2,
	// but even if you do, there's little advantage of S0 B->A in most cases.
	// In S1, it only makes sense if there are so few tags you can read them
	// before they time out and revert themselves B->A anyway.
	// So this makes the most sense in either S2 or S3,
	// particularly when attempting to read a large, mostly static population.
	//
	// Impinj says its useful for "Low-to-medium tag count,
	// low-throughput [...] repeated tag observation".
	impjDualTarget = impinjSearchMode(2)

	// impjDualTargetWithReset is Impinj's "Dual Target Inventory with Reset".
	// Despite the name, this mode only targets state A in the given session.
	// Once it's quiet, it uses a Select command to flip tags' states B->A.
	//
	// Note that as far as Impinj is concerned,
	// "Reset" means "put the session flag into state B".
	// In "SingleTargetReset", the "Reset" is just a normal Query command,
	// whereas here in "DualTargetReset", the "Reset" is a Select command.
	// Sending a Select command before an inventory round
	// takes a long time relative singulating a single tag,
	// but since it only has to be done once at the beginning of the round,
	// that time is amortized over the tag population,
	// so it's far more efficient for a large population.
	// On the other hand, if a tag doesn't "hear" the Select command,
	// its session flag will remain in state B,
	// and it won't be inventoried during that round.
	//
	// In standard LLRP with State Aware filtering,
	// you can achieve this by targeting A
	// and using a Filter that targets all tags and clears matches,
	// or one that targets no tags and sets non-matches.
	// You can put these in the same AISpec,
	// in which case the Reader should send the Select before every inventory round,
	// or you can put the Filter in an AISpec that times out quickly,
	// and configure the Stop trigger on the "main" spec
	// so that it only stops after some time without seeing new tags.
	//
	// They say it's good for "High tag count,
	// high-throughput [with] repeated observation".
	impjDualTargetWithReset = impinjSearchMode(6)
)

// ExtractRSSI returns the RSSI value from TagReportData, if present.
//
// If the report includes a Custom Impinj RSSI parameter, it returns that.
// Because those values are dBm x100, it converts it to dBm (by dividing by 100),
// and hence the returned value is a floats instead of an int.
func (rt *TagReportData) ExtractRSSI() (rssi float64, ok bool) {
	for _, c := range rt.Custom {
		if c.Is(PENImpinj, ImpinjEnablePeakRSSI) && len(c.Data) == 2 {
			return float64(binary.BigEndian.Uint16(c.Data)) / 100.0, true // dBm x100
		}
	}

	if rt.PeakRSSI != nil {
		rssi = float64(*rt.PeakRSSI)
		ok = true
	}
	return
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
	}

	return
}

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
