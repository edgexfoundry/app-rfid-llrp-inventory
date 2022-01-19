//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

//go:generate stringer -type=VendorPEN,ImpinjModel -output vendors_string.go

package llrp

// VendorPEN are constants that represent common known Private Enterprise Numbers
type VendorPEN uint32

const (
	PENImpinj = VendorPEN(25882)
	PENAlien  = VendorPEN(17996)
	PENZebra  = VendorPEN(10642)
)

// ImpinjModel are mappings for known Impinj model numbers to their model names.
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

// CustomParamSubtype is a base type for all custom param subtypes
type CustomParamSubtype = uint32

// ImpinjParamSubtype are constant definitions of Impinj specific custom functionality
type ImpinjParamSubtype = CustomParamSubtype

const (
	ImpinjEnablePeakRSSI           = ImpinjParamSubtype(53)
	ImpinjPeakRSSI                 = ImpinjParamSubtype(57)
	ImpinjTagReportContentSelector = ImpinjParamSubtype(50)
	ImpinjSearchMode               = ImpinjParamSubtype(23)
)

// impinjSearchMode is like a really limited version of standard state-aware filtering
// with added ambiguity about what C1G2 commands the Reader might send.
type impinjSearchMode = uint16

const (
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
