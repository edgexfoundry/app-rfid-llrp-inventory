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

	// impSearchQueryAtoB Impinj calls "Single Target".
	// It sets the Target field in Queries to A,
	// but there's no indication what it uses for the SL flag.
	//
	// It has the effect of setting singulated tags' Session flag to B.
	// In S2 and S3, tags remain quiet once singulated as long as they're powered.
	// In S1, they'll fall back after the persistence timeout (500ms-5s),
	// so as long as the population small enough to read before that timeout,
	// this has the effect of "spreading" their observations through the read window.
	// In S0, the flag resets so quickly, this is unlikely to have much impact.
	impSearchQueryAtoB = impinjSearchMode(1)

	// impSearchQueryBtoA Impinj calls "Single Target Reset".
	// This is just a Query with the Target set to B instead of A.
	impSearchQueryBtoA = impinjSearchMode(5)

	// impSearchQueryAtoBSupMonzaS1 is maybe the reason Impinj has "Search Modes".
	// They call this one "Single Target Inventory with Suppression (aka TagFocus)",
	// it is just the above QueryAtoB mode, but sends a command to Impinj Monza tags
	// to refresh their S1 flag persistence:
	// https://platform.impinj.com/indy/itk/latest/Glossary/Glossary.html#term-tagfocus
	//
	// Importantly, it only makes sense if used with Session 1,
	// and only if you've got mostly Impinj Monza tags.
	// Otherwise it's just the same as querying A->B, but probably slower.
	//
	// The S1 flag resets itself B->A after 500ms-5s, unless refreshed,
	// which can be done with standard LLRP commands
	// if a Reader supports State Aware filtering,
	// but does require more than one AISpec.
	//
	// It's unclear how Impinj is issuing custom commands to its Monza tags
	// while still complying with part 2.3.4 of the Gen2 standard:
	// "An Interrogator shall issue a custom command only after (1) singulating a Tag,
	// and (2) reading (or having prior knowledge of) the Tag['s] TID [...]
	// A custom command shall not solely duplicate the functionality
	// of any mandatory or optional command defined in this protocol".
	// That should slow down the process considerably,
	// but since it doesn't, maybe they're just sending a Select
	// that filters on their TID but the tag interprets it a special way?
	impSearchQueryAtoBSupMonzaS1 = impinjSearchMode(3)

	// impSearchQueryAtoBtoA Impinj calls "Dual Target Inventory" and says it:
	//
	//   "Inventories tags in state A, transitioning the tags to state B.
	//    Inventories tags in state B, transitioning the tags to state [A, sic]".
	//
	// This should require multiple Queries, repeated in a loop:
	// the first with Target set to B, repeated until there are few/no observations,
	// and a second with Target set to A, likewise repeated until quiet.
	// The process repeats indefinitely.
	// In S0, you're likely to never move from step 1 to step 2,
	// but even if you do, there's little advantage of S0 B->A in most cases.
	// In S1, it only makes sense if there are so few tags you can read them
	// before they time out and revert themselves B->A anyway.
	// So this makes the most sense in either S2 or S3,
	// particularly when attempting to read a large, mostly static population.
	//
	// Impinj says its useful for "Low-to-medium tag count,
	// low-throughput [...] repeated tag observation".
	impSearchQueryAtoBtoA = impinjSearchMode(2)

	// impSearchSelToAQueryAtoB Impinj calls "Dual Target Inventory with Reset",
	// and says is good for "High tag count, high-throughput [with] repeated observation".
	//
	// This mode sends Queries with Target A until it's quiet,
	// then sends a Select command to flip the session B->A.
	// In standard LLRP with State Aware filtering,
	// this is just two AISpecs:
	// the first with a Filter for the Select command
	// which times out quickly (or searches for a single tag, etc.)
	// followed by a second AISpec that just inventories Target A.
	impSearchSelToAQueryAtoB = impinjSearchMode(6)
)

func (rt *TagReportData) ExtractRSSI() (rssi float64, ok bool) {
	for _, c := range rt.Custom {
		if c.Is(PENImpinj, ImpinjEnablePeakRSSI) && len(c.Data) == 2 {
			return float64(binary.BigEndian.Uint16(c.Data) / 100.0), true // dBm x100
		}
	}

	if rt.PeakRSSI != nil {
		rssi = float64(*rt.PeakRSSI)
		ok = true
	}
	return
}

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
