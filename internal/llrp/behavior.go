//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

// Package llrp implements higher-level logic atop an LLRP Reader.
// This package converts <LLRP Reader Info, Desired Behavior> to LLRP messages & parameters.
package llrp

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"sort"
	"strings"
)

// Behavior is a high-level description of desired Reader operation.
//
// LLRP Readers vary wildly in their capabilities;
// some Behavior characteristics cannot be well-mapped to all Readers.
type Behavior struct {
	GPITrigger    *GPITrigger    `json:",omitempty"`
	ImpinjOptions *ImpinjOptions `json:",omitempty"`

	ScanType    ScanType
	Duration    Millisecs32 // 0 = repeat forever
	Power       PowerTarget
	Frequencies []Kilohertz `json:",omitempty"` // ignored in Hopping regions
}

type GPITrigger struct {
	Port    uint16
	Event   bool
	Timeout Millisecs32 `json:",omitempty"`
}

// ImpinjOptions control behaviors that will only apply to Impinj Readers,
// usually because they make use of some custom behavior only implemented there.
type ImpinjOptions struct {
	// SuppressMonza enables Impinj's "TagFocus" feature.
	//
	// When enabled, when a Behavior uses S1,
	// the Reader refreshes Monza tags' S1 flag B state.
	// Monza tags should be inventoried one time when they enter the FoV
	// This reduces repeated observations of a tag that stays within an antenna's FoV.
	// Note that this only works on Impinj Monza tags;
	// other tags should revert their S1 flag normally,
	// and thus will get re-inventoried every so often,
	// regardless of movement in and out of antennas' Fields of View.
	SuppressMonza bool
}

// PowerTarget specifies a target power for the Reader to push through the antenna.
//
// It does not account for losses or gains,
// nor does it make any guarantees about max radiated power
// or compliance with local regulatory requirements.
// The power is assumed valid at all Frequencies
type PowerTarget struct {
	Max MillibelMilliwatt
}

type ScanType int

const (
	ScanFast = ScanType(iota)
	ScanNormal
	ScanDeep
)

var (
	ErrMissingCapInfo = fmt.Errorf("missing capability information")
	ErrUnsatisfiable  = fmt.Errorf("behavior cannot be satisfied")
)

func errMissingCapInfo(name string, path ...string) error {
	if len(path) != 0 {
		return errors.Wrapf(ErrMissingCapInfo, "missing LLRP %s from %s",
			name, strings.Join(path, "."))
	}
	return errors.Wrapf(ErrMissingCapInfo, "missing LLRP %s", name)
}

// BasicDevice holds details of an LLRP device
// and uses those details to determine how best to satisfy desired behaviors.
//
// In particular, it maintains lists of LLRP capabilities,
// such as UHF RF Modes and device power levels.
// Much of the information it tracks mirrors the device's capabilities,
// but it preprocesses some of it to simplify later device command generation.
type BasicDevice struct {
	// connected   time.Time
	modes       []UHFC1G2RFModeTableEntry
	pwrMinToMax []TransmitPowerLevelTableEntry
	freqInfo    FrequencyInformation

	// report is the collection of information we want expect a Reader to report.
	// LLRP has a data compression "feature" that allows Readers to omit some parameters
	// if the value hasn't changed "since the last time it was sent".
	report TagReportContentSelector
	// lastData is the value of tag parameter the last time it was reported.
	lastData TagReportData

	nGPIs, nFreqs uint16
	nSpecsPerRO   uint32
	allowsHop     bool
	stateAware    bool
}

// ImpinjDevice embeds BasicDevice to provide some Impinj-specific Behavior implementations.
//
// Impinj isn't compliant with the following elements of the LLRP standard:
// - UHF Modes incorrectly report BLF (in Hz) instead of BDR (in bps).
// - UHF Modes include "Autoset" modes with IDs > 1000,
//   for which the parameter values are made up;
//   the actual mode used is one of the non-Autoset modes,
//   but the Reader interprets the given ModeID as a hint
//   for it to choose which of those it thinks is best.
// - Truncate actions during Select (i.e., C1G2Filter.T) are not supported and must be 0.
// - Per-antenna configurations are not compliant in general,
//   but you can set transmit power and receive sensitivity per-antenna.
//
// Additionally, the following limitations have been observed:
// - If Hopping is True, we've observed only a single HopTable.
// - Depending on the firmware and specific Reader type, 2-5 C1G2 Filters are available.
// - There seem to be 5 real UHF Modes, though which are available
//   is limited by Reader version and Firmware.
//   All of them use Multi-Reader or Dense-Reader spectral masks,
//   a DR of 64/3, and do not permit Tari selection (min==max):
//   - Mode0 is their fastest, at 640kbps using Tari 6.25us, PIER 1.5, and FM0.
//   - Mode1 they call "Hybrid"; it's the same as Mode0,
//     but uses Miller2 backscatter encoding, so the BDR is 320kbps.
//   - Mode2 is for "Dense" environments and has a BDR of 68.5kbps.
//   - Mode3 is for even denser environments, with the slowest BDR at 21.25kbps.
//     They call it MaxMiller because it uses Miller8 backscatter encoding.
//   - They don't support State Aware Filtering, at least not directly.
//     There is a custom parameter for "Search Mode" which essentially does it.
type ImpinjDevice struct {
	BasicDevice
}

func NewBasicDevice(c *GetReaderCapabilitiesResponse) (*BasicDevice, error) {
	if c == nil || c.LLRPCapabilities == nil || c.GeneralDeviceCapabilities == nil ||
		c.RegulatoryCapabilities == nil || c.C1G2LLRPCapabilities == nil {
		return nil, errMissingCapInfo("capabilities")
	}

	regCap := c.RegulatoryCapabilities
	if regCap == nil || regCap.UHFBandCapabilities == nil ||
		len(regCap.UHFBandCapabilities.TransmitPowerLevels) == 0 {
		return nil, errMissingCapInfo("power levels",
			"RegulatoryCapabilities", "UHFBandCapabilities", "TransmitPowerLevels")
	}

	modes := regCap.UHFBandCapabilities.C1G2RFModes.UHFC1G2RFModeTableEntries
	if len(modes) == 0 {
		return nil, errMissingCapInfo("RF modes",
			"RegulatoryCapabilities", "UHFBandCapabilities",
			"C1G2RFModes", "UHFC1G2RFModeTableEntries")
	}

	copyModes := make([]UHFC1G2RFModeTableEntry, len(modes))
	copy(copyModes, modes)

	var nFreqs uint16
	freqInfo := regCap.UHFBandCapabilities.FrequencyInformation
	if freqInfo.Hopping {
		if len(freqInfo.FrequencyHopTables) == 0 {
			return nil, errMissingCapInfo("frequency table",
				"RegulatoryCapabilities", "UHFBandCapabilities",
				"FrequencyInformation", "FrequencyHopTables")
		}

		// Array fields in binary LLRP messages can't be longer than a uint16,
		// so this can only trigger if it didn't come from an LLRP message;
		// since we can't use the value, at least let the programmer know
		// they've created an illegal situation somehow.
		if len(freqInfo.FrequencyHopTables[0].Frequencies) > (1 << 16) {
			panic("impossible frequency table length")
		}
		nFreqs = uint16(len(freqInfo.FrequencyHopTables[0].Frequencies))
	} else {
		if freqInfo.FixedFrequencyTable == nil || len(freqInfo.FixedFrequencyTable.Frequencies) == 0 {
			return nil, errMissingCapInfo("frequency table",
				"RegulatoryCapabilities", "UHFBandCapabilities",
				"FrequencyInformation", "FixedFrequencyTable", "Frequencies")
		}

		// See notes above about why this precaution is very unlikely to trigger.
		if len(freqInfo.FixedFrequencyTable.Frequencies) > (1 << 16) {
			panic("impossible frequency table length")
		}
		nFreqs = uint16(len(freqInfo.FixedFrequencyTable.Frequencies))
	}

	genCap := c.GeneralDeviceCapabilities
	if genCap == nil {
		return nil, errMissingCapInfo("GPI count", "GeneralCapabilities", "GPIOCapabilities")
	}

	llrpCap := c.LLRPCapabilities

	// Copy & sort the power level entries by power level, min to max;
	// We make a copy of the list because:
	// - we may need to change the item order
	// - we need to know the values won't change
	// - we don't want to prevent the GC from reclaiming the *Capabilities memory
	//
	// Since we need to iterate to copy, we check if it's already sorted, too.
	tpl := regCap.UHFBandCapabilities.TransmitPowerLevels
	isSorted := true
	last := tpl[0].TransmitPowerValue // above checks len >0
	pwrLvls := make([]TransmitPowerLevelTableEntry, len(tpl))
	for i, entry := range tpl {
		pwrLvls[i] = entry

		// if this entry is less than the previous, it's not sorted in ascending order
		if entry.TransmitPowerValue < last {
			isSorted = false
		} else {
			last = entry.TransmitPowerValue
		}
	}

	if !isSorted {
		sort.Slice(pwrLvls, func(i, j int) bool {
			return pwrLvls[i].TransmitPowerValue < pwrLvls[j].TransmitPowerValue
		})
	}

	return &BasicDevice{
		modes:       copyModes,
		pwrMinToMax: pwrLvls,
		nFreqs:      nFreqs,
		nGPIs:       genCap.GPIOCapabilities.NumGPIs,
		freqInfo:    freqInfo,
		allowsHop:   freqInfo.Hopping,
		nSpecsPerRO: llrpCap.MaxSpecsPerROSpec,
		stateAware:  llrpCap.CanDoTagInventoryStateAwareSingulation,
		lastData: TagReportData{
			ROSpecID:                 new(ROSpecID),
			SpecIndex:                new(SpecIndex),
			InventoryParameterSpecID: new(InventoryParameterSpecID),
			AntennaID:                new(AntennaID),
			PeakRSSI:                 new(PeakRSSI),
			ChannelIndex:             new(ChannelIndex),
			FirstSeenUTC:             new(FirstSeenUTC),
			LastSeenUTC:              new(LastSeenUTC),
			TagSeenCount:             new(TagSeenCount),
		},
	}, nil
}

func NewImpinjDevice(c *GetReaderCapabilitiesResponse) (*ImpinjDevice, error) {
	bd, err := NewBasicDevice(c)
	if err != nil {
		return nil, err
	}

	// Correct Impinj's buggy mode table
	fixed := make([]UHFC1G2RFModeTableEntry, 0, len(bd.modes)/2)
	for _, m := range bd.modes {
		if m.ModeID >= 1000 { // the values for these modes are meaningless
			continue
		}

		// They're reporting BLF (in Hz) instead of BDR (in bps).
		m.BackscatterDataRate >>= m.Modulation
		fixed = append(fixed, m)
	}
	bd.modes = fixed

	return &ImpinjDevice{BasicDevice: *bd}, nil
}

func (d *BasicDevice) NewConfig() *SetReaderConfig {
	return &SetReaderConfig{
		ResetToFactoryDefaults: true,
		ROReportSpec: &ROReportSpec{
			Trigger: NTagsOrAIEnd,
			N:       1,

			TagReportContentSelector: TagReportContentSelector{
				EnableLastSeenTimestamp: true,
				EnableAntennaID:         true,
				EnablePeakRSSI:          true,
			},
		},
	}
}

func (d *ImpinjDevice) NewConfig() *SetReaderConfig {
	conf := d.BasicDevice.NewConfig()

	conf.ROReportSpec.Custom = append(conf.ROReportSpec.Custom, Custom{
		VendorID: uint32(PENImpinj),
		Subtype:  ImpinjTagReportContentSelector,
		Data:     impinjEnableBool16(ImpinjEnablePeakRSSI),
	})
	return conf
}

// impinjEnableBool16 returns the encoding of a Custom parameter
// that consists only of a boolean value represented as a uint16,
// since Impinj uses them often as sub-parameters of their own Custom parameters.
func impinjEnableBool16(subtype ImpinjParamSubtype) []byte {
	const (
		pen0 = uint8(0xff & (uint32(PENImpinj) >> (8 * (3 - iota))))
		pen1
		pen2
		pen3
	)

	return []byte{
		0x03, 0xff, 0, 14, // param type & length (incl. header)
		pen0, pen1, pen2, pen3,
		uint8(subtype >> 24), uint8(subtype >> 16), uint8(subtype >> 8), uint8(subtype),
		0, 1, // data (uint16, 0=disabled, 1=enabled)
	}
}

// ProcessTagReport processes what it expects is the most recent TagReportData.
//
// Currently, it fills in ambiguous nil values in the report data,
// according to LLRP's rules for "efficient" tag reporting.
func (d *BasicDevice) ProcessTagReport(tags []TagReportData) {
	d.fillAmbiguousNil(tags)
}

// ProcessTagReport does nothing for Impinj Readers,
// as they say they always send all parameters in every report.
// Hopefully that is true.
func (d *ImpinjDevice) ProcessTagReport(_ []TagReportData) {}

// fillAmbiguousNil handles the worst feature of LLRP: ambiguous nil parameters.
//
// Specifically, it fills in tag data parameters that weren't reported
// because they match the last reported value of the same type.
// This can only be done correctly if you know enough context,
// so this method is assuming we're using a consistent set of reporting parameters,
// and tag reports are processed in full, in order.
// It also skips Uptimes and AccessSpecID parameters.
//
// LLRP allows Readers to use `nil` to mean both "not enabled" and "hasn't changed".
// The Client "knows" which it is because they know if the value were enabled,
// and they know the most recent value of each optional parameter ever received.
//
// You can't disable this behavior.
// You can't even query a Reader to know if it's something the Reader supports.
//
// As a result, the Clients must track:
// - the most-recently-received value of every optional parameter
// - the reporting parameters of any ROSpec
//   for which it might still receive tag reports;
//   note that it's legal to delete an ROSpec before requesting its data
// - the default reporting parameters at any point they were changed,
//   if there was defined at that time an enabled ROSpec that used the defaults
// - the ROSpecIDs and start/stop timestamps of any ROSpec
//   for which it might still receive tag reports;
//   since this is itself an optional parameter,
//   there are several ways to configure an LLRP Reader
//   such that it is impossible to disambiguate nil parameters.
//
// Here's a direct quote from the LLRP Spec explaining how it works:
//
//		This report parameter is generated per tag per accumulation scope[*].
//		The only mandatory portion of this parameter is the EPCData parameter.
//		If there was an access operation performed on the tag,
//		the results of the OpSpecs are mandatory in the report.
//		The other sub-parameters in this report are optional.
//		LLRP provides three ways to make the tag reporting efficient:
//
//		(i) Allow parameters to be enabled or disabled via TagReportContentSelector in TagReportSpec.
//		(ii) If an optional parameter is enabled, and is absent in the report,
//		the Client SHALL assume that the value is identical
//		to the last parameter of the same type received.
//		For example, this allows the Readers to not send a parameter in the report
//		whose value has not changed since the last time it was sent by the Reader.
//
// [*] This is just saying you get a TagReportData parameter
//     for each EPC and unique combination of OpSpec result or matched IDs.
//     Report accumulation also affects the reporting of
//     timestamps, RSSI, the channel index, and number of observations.
//
func (d *BasicDevice) fillAmbiguousNil(tags []TagReportData) {
	for i := range tags {
		tag := &tags[i]
		if d.report.EnableROSpecID {
			if tag.ROSpecID == nil {
				tag.ROSpecID = new(ROSpecID)
				*tag.ROSpecID = *d.lastData.ROSpecID
			} else {
				*d.lastData.ROSpecID = *tag.ROSpecID
			}
		}

		if d.report.EnableSpecIndex {
			if tag.SpecIndex == nil {
				tag.SpecIndex = new(SpecIndex)
				*tag.SpecIndex = *d.lastData.SpecIndex
			} else {
				*d.lastData.SpecIndex = *tag.SpecIndex
			}
		}

		if d.report.EnableInventoryParamSpecID {
			if tag.InventoryParameterSpecID == nil {
				tag.InventoryParameterSpecID = new(InventoryParameterSpecID)
				*tag.InventoryParameterSpecID = *d.lastData.InventoryParameterSpecID
			} else {
				*d.lastData.InventoryParameterSpecID = *tag.InventoryParameterSpecID
			}
		}

		if d.report.EnableAntennaID {
			if tag.AntennaID == nil {
				tag.AntennaID = new(AntennaID)
				*tag.AntennaID = *d.lastData.AntennaID
			} else {
				*d.lastData.AntennaID = *tag.AntennaID
			}
		}

		if d.report.EnablePeakRSSI {
			if tag.PeakRSSI == nil {
				tag.PeakRSSI = new(PeakRSSI)
				*tag.PeakRSSI = *d.lastData.PeakRSSI
			} else {
				*d.lastData.PeakRSSI = *tag.PeakRSSI
			}
		}

		if d.report.EnableChannelIndex {
			if tag.ChannelIndex == nil {
				tag.ChannelIndex = new(ChannelIndex)
				*tag.ChannelIndex = *d.lastData.ChannelIndex
			} else {
				*d.lastData.ChannelIndex = *tag.ChannelIndex
			}
		}

		if d.report.EnableFirstSeenTimestamp {
			if tag.FirstSeenUTC == nil {
				tag.FirstSeenUTC = new(FirstSeenUTC)
				*tag.FirstSeenUTC = *d.lastData.FirstSeenUTC
			} else {
				*d.lastData.FirstSeenUTC = *tag.FirstSeenUTC
			}
		}

		if d.report.EnableLastSeenTimestamp {
			if tag.LastSeenUTC == nil {
				tag.LastSeenUTC = new(LastSeenUTC)
				*tag.LastSeenUTC = *d.lastData.LastSeenUTC
			} else {
				*d.lastData.LastSeenUTC = *tag.LastSeenUTC
			}
		}

		if d.report.EnableTagSeenCount {
			if tag.TagSeenCount == nil {
				tag.TagSeenCount = new(TagSeenCount)
				*tag.TagSeenCount = *d.lastData.TagSeenCount
			} else {
				*d.lastData.TagSeenCount = *tag.TagSeenCount
			}
		}
	}
}

// Transmit returns a legal llrp.RFTransmitter value.
func (d *BasicDevice) Transmit(b Behavior) (*RFTransmitter, error) {
	// First, find the highest power at or below the Target.
	pwrIdx, pwr := d.findPower(b.Power.Max)
	if pwr > b.Power.Max {
		return nil, errors.Wrapf(ErrUnsatisfiable,
			"target power (%.2f dBm) is lower than the lowest supported (%.2f dBm)",
			float32(b.Power.Max)/100.0, float32(pwr)/100.0)
	}

	// In hopping regulatory regions, we assume the power is legal for all frequencies.
	// This also assumes the first HopTableID is acceptable/worth using,
	// which may or may not be true.
	if d.allowsHop {
		return &RFTransmitter{
			HopTableID:         uint16(d.freqInfo.FrequencyHopTables[0].HopTableID),
			TransmitPowerIndex: pwrIdx,
		}, nil
	}

	// For non-hopping regions, we need to find a frequency that permits this power level.
	for _, permitted := range b.Frequencies {
		for i, f := range d.freqInfo.FixedFrequencyTable.Frequencies {
			if permitted == f {
				// +1 because channel indices are 1-based in LLRP
				return &RFTransmitter{ChannelIndex: uint16(i + 1)}, nil
			}
		}
	}

	return nil, errors.Wrapf(ErrUnsatisfiable,
		"no frequency permits the desired power level (%.2f dBm)",
		float32(b.Power.Max)/100.0)
}

// findPower returns the device's best match to a given power level,
// suitable for use as the RFTransmitter index in AntennaConfigurations.
//
// The returned power level and its respective index
// is the highest supported power level less than or equal to the target;
// if the target is less than even the lowest supported power level,
// then this returns the lowest power level and its respective index,
// so you should check the value upon return if a higher level is never suitable.
//
// This panics if there is not at least one power value.
func (d *BasicDevice) findPower(target MillibelMilliwatt) (tableIdx uint16, value MillibelMilliwatt) {
	// sort.Search returns the smallest index i at which f(i) is true,
	// or the list len if the result is always false.
	// This requires the list is sorted (in our case, in ascending order).
	pwrIdx := sort.Search(len(d.pwrMinToMax), func(i int) bool {
		return d.pwrMinToMax[i].TransmitPowerValue >= target
	})

	var t TransmitPowerLevelTableEntry
	if pwrIdx == 0 {
		t = d.pwrMinToMax[0]
	} else if pwrIdx < len(d.pwrMinToMax) && d.pwrMinToMax[pwrIdx].TransmitPowerValue == target {
		// The power exactly matches one of the Reader's available power settings.
		t = d.pwrMinToMax[pwrIdx]
	} else {
		// The index represents a power value greater than our target,
		// so instead return one less than the target.
		// This also covers the case where the target power
		// is greater than any available power setting,
		// as in that case pwrIdx = len(list),
		// and we want to return the max power, which is at len(list)-1
		t = d.pwrMinToMax[pwrIdx-1]
	}

	return t.Index, t.TransmitPowerValue
}

// findBestMode returns the best RF Mode for the given environment density.
//
// If the number of nearby Readers is unknown, use 0.
// This returns both the best RF Mode entry as well as its 0-index within the slice.
func (d *BasicDevice) findBestMode(nReaders uint) (bestIdx int, mode UHFC1G2RFModeTableEntry) {
	const dense = 0.5 // EPC spec implies >50% is about where "multi" becomes "dense"
	var maskTarget SpectralMaskType
	switch nReaders {
	case 0:
		maskTarget = SpectralMaskUnknown
	case 1:
		maskTarget = SpectralMaskSingleInterrogator
	default:
		density := float64(nReaders) / float64(d.nFreqs)
		if nReaders >= uint(d.nFreqs) || density > dense {
			maskTarget = SpectralMaskDenseInterrogator
		} else {
			maskTarget = SpectralMaskMultiInterrogator
		}
	}

	// Start by only considering modes at least as high as our density,
	// as a higher data rate is pretty useless if interference skyrockets.
	// If there's no mode at or above the mask density,
	// drop it down and try again; this ensures we always return a mode.
	for {
		mID, ok := d.fastestAt(maskTarget)
		if ok {
			bestIdx = mID
			break
		}

		// This should be impossible, but check just in case.
		if maskTarget == 0 {
			panic("no modes")
		}
		maskTarget--
	}

	return bestIdx, d.modes[bestIdx]
}

// fastestAt returns the index of the RF Mode with the highest likely throughput
// at or above the given density.
// If there are no modes at or above the given density mask,
// the returned "ok" value is false, and bestIdx is undefined.
//
// The returned bestIdx is the 0-indexed Go slice index,
// not the LLRP-defined ModeID of the relevant mode.
// There must be at least one mode in the mode table.
// If the input mask level is 0 ("Unknown"),
// then "ok" will be true and bestIdx will be valid.
//
// LLRP (and this code) abstracts reader density via the mode's "SpectralMask"
// (the name relates to how Readers make use of the available channel spectrum).
// A higher mask level implies a more dense Reader environment:
// one in which most or all available frequency freqInfo are occupied.
// Minimizing collisions requires frequency-division multiplexing,
// preferably by choosing backscattered link frequencies and modulations
// that permit guardbands between the carrier waves and backscattered sidebands.
// More information can be found in Appendix G of the
// EPC Radio-Frequency Identity Protocols Generation-2 UHF RFID Standard.
func (d *BasicDevice) fastestAt(mask SpectralMaskType) (bestIdx int, ok bool) {
	// During singulation, tags only backscatter about 150 bits.
	// At low BDRs, the backscatter time indeed dominates singulation,
	// but at higher BDRs, the forward link can make up to a 3x difference.
	//
	// Roughly speaking, RTcal is an OK approximation of the forward link
	// and BDR is an OK approximation of the backward link.
	// This scales them relative their best possible values
	// and calculates a score as a weighted average of those value.
	//
	// fwdLinkBias determines how much importance
	// we place on the forward vs backward links.
	// 0.5 gives equal weight to the forward and backward links,
	// effectively the special case of simply averaging them.
	// It must be between 0 and 1.
	const fwdLinkBias = 0.5
	const bestRTcal, bestBDR = 15625000, 640000
	var bestScore float64 // lower is better

	for i, m := range d.modes {
		if m.SpectralMask < mask { // skip modes with too much interference
			continue
		}

		fwdLink := bestRTcal / (float64(m.MinTariTime) * float64(1000+m.PIERatio))
		bwdLink := bestBDR / float64(m.BackscatterDataRate)
		score := (fwdLinkBias * fwdLink) + ((1 - fwdLinkBias) * bwdLink)
		if !ok || score < bestScore {
			bestScore = score
			bestIdx = i
		}

		ok = true
	}

	return bestIdx, ok
}

type TagMobility uint16

const (
	tagMobilityUnknown = TagMobility(0)
	tagsAreStatic      = TagMobility(500)
	tagsMayMove        = TagMobility(5000)
	tagsAreInMotion    = TagMobility(10000)
)

// Environment describes the expected operating environment.
// For unknown values, set the field to its zero value.
type Environment struct {
	NumNearbyReaders uint
	PopulationSize   uint16
	Mobility         TagMobility
}

// NewROSpec returns a new llrp.ROSpec to achieve the Behavior within the Environment.
func (d *BasicDevice) NewROSpec(b Behavior, e Environment) (*ROSpec, error) {
	if b.GPITrigger != nil && (b.GPITrigger.Port == 0 ||
		d.nGPIs == 0 || b.GPITrigger.Port > d.nGPIs) {
		return nil, errors.Wrapf(ErrUnsatisfiable,
			"behavior uses a GPI Trigger with invalid Port "+
				"(%d not in [1, %d])", b.GPITrigger.Port, d.nGPIs)
	}

	transmit, err := d.Transmit(b)
	if err != nil {
		return nil, err
	}

	mIdx, best := d.findBestMode(e.NumNearbyReaders)
	tari := d.modes[mIdx].MinTariTime
	canDualTarget := d.stateAware && d.nSpecsPerRO >= 2

	// This query only applies if the Reader supports state aware actions.
	query := &C1G2SingulationControl{
		InvAwareAction: new(C1G2TagInventoryStateAwareSingulationAction),
	}

	// Our basic AISpec targets all antennas and runs forever.
	aiSpecs := []AISpec{{
		AntennaIDs: []AntennaID{0},
		InventoryParameterSpecs: []InventoryParameterSpec{{
			InventoryParameterSpecID: 1,
			AirProtocolID:            AirProtoEPCGlobalClass1Gen2,
			AntennaConfigurations: []AntennaConfiguration{{
				AntennaID:     0,
				RFTransmitter: transmit,
				C1G2InventoryCommand: &C1G2InventoryCommand{
					TagInventoryStateAware: d.stateAware,
					RFControl: &C1G2RFControl{
						RFModeID: uint16(best.ModeID),
						Tari:     uint16(tari),
					},

					SingulationControl: query,
				},
			}},
		}},
	}}

	// Grab the pointer to the first InventoryCommand
	invCmd := aiSpecs[0].InventoryParameterSpecs[0].
		AntennaConfigurations[0].C1G2InventoryCommand

	switch b.ScanType {
	case ScanFast:
		invCmd.Filters = nil // remove the filter

		// For a Fast scan, search for tags with S0 in State A.
		// S0 reverts on its own when not powered.
		*query = C1G2SingulationControl{
			Session:        0,
			TagPopulation:  500,
			TagTransitTime: 500,
			InvAwareAction: &C1G2TagInventoryStateAwareSingulationAction{
				SessionState: SessionStateA,
				SLState:      SLStateDeasserted,
			},
		}

	case ScanNormal:
		// For a Normal scan, search for tags with S1 in State A.
		// Inventorying S1 allows a lower Q value without collisions,
		// as over time tags spread out in the persistence time window.
		// If the population is smaller than (S1 persistence) * (read rate),
		// then eventually each round only one tag is eligible for singulation
		// (i.e., only a single tag's S1 will have fallen back to A).
		*query = C1G2SingulationControl{
			Session:        1,
			TagPopulation:  1000,
			TagTransitTime: 5000,
			InvAwareAction: &C1G2TagInventoryStateAwareSingulationAction{
				SessionState: SessionStateA,
				SLState:      SLStateDeasserted,
			},
		}

	case ScanDeep:
		// If we can't control session flags directly
		// or if we can only have a single AISpec,
		// we use a Filter with a mask that matches no tags
		// with an action that "Selects" non-matches and "Unselects" matches,
		// then set the SingulationControl to target S2.
		// The only reasonable way for a Reader to implement that
		// is with a Select for S2 B->A and a Query for target S2 in state A.
		// Our choice of Filter effectively matches all tags,
		// but minimizes the amount of data the Reader actually sends.
		//
		// If we have enough controls to do so, just use two Queries:
		// first inventory S2 A->B until it's quiet for 500ms or more,
		// then inventory S2 B->A until it's quiet for 500ms or more.
		if !d.stateAware || d.nSpecsPerRO == 1 {
			action := C1G2TagInventoryStateUnawareFilterAction(UnawareSelectMKeepUClear)

			invCmd.Filters = []C1G2Filter{{
				TruncateAction: FilterActionDoNotTruncate,
				TagInventoryMask: C1G2TagInventoryMask{
					MemoryBank: 1,
				},
				UnawareFilterAction: &action,
			}}

			*query = C1G2SingulationControl{
				Session:        2,
				TagPopulation:  3000,
				TagTransitTime: 10000,
				InvAwareAction: &C1G2TagInventoryStateAwareSingulationAction{
					SessionState: SessionStateB,
					SLState:      SLStateDeasserted,
				},
			}

		}

		if !canDualTarget {
			break
		}

		aiSpecs = make([]AISpec, 2)
		for i := range aiSpecs {
			sessionState := SessionStateB
			if i&1 == 0 {
				sessionState = SessionStateA
			}

			aiSpecs[i] = AISpec{
				AntennaIDs: []AntennaID{0},
				StopTrigger: AISpecStopTrigger{
					Trigger: AIStopTriggerTagObservation,
					TagObservationTrigger: &TagObservationTrigger{
						Trigger: TagObsTriggerNoNewAfterT,
						T:       500,
					},
				},

				InventoryParameterSpecs: []InventoryParameterSpec{{
					InventoryParameterSpecID: uint16(i + 1),
					AirProtocolID:            AirProtoEPCGlobalClass1Gen2,
					AntennaConfigurations: []AntennaConfiguration{{
						AntennaID:     0,
						RFTransmitter: transmit,
						C1G2InventoryCommand: &C1G2InventoryCommand{
							TagInventoryStateAware: true,
							RFControl: &C1G2RFControl{
								RFModeID: uint16(best.ModeID),
								Tari:     uint16(tari),
							},
							SingulationControl: &C1G2SingulationControl{
								Session:        2,
								TagPopulation:  500,
								TagTransitTime: 500,
								InvAwareAction: &C1G2TagInventoryStateAwareSingulationAction{
									SessionState: sessionState,
									SLState:      SLStateDeasserted,
								},
							},
						},
					}},
				}},
			}
		}
	}

	if e.PopulationSize != 0 {
		query.TagPopulation = e.PopulationSize
	}

	if e.Mobility != tagMobilityUnknown {
		query.TagTransitTime = Millisecs32(e.Mobility)
	}

	spec := &ROSpec{
		ROSpecID:       1, // May be overridden, but better to ensure it's not 0.
		ROBoundarySpec: b.Boundary(),
		AISpecs:        aiSpecs,
	}

	return spec, nil
}

// NewROSpec returns a new llrp.ROSpec to achieve the Behavior within the Environment
// with some aid of Impinj-specific LLRP vendor extensions.
func (d *ImpinjDevice) NewROSpec(b Behavior, e Environment) (*ROSpec, error) {
	if b.GPITrigger != nil && (b.GPITrigger.Port == 0 ||
		d.nGPIs == 0 || b.GPITrigger.Port > d.nGPIs) {
		return nil, errors.Wrapf(ErrUnsatisfiable,
			"behavior uses a GPI Trigger with invalid Port "+
				"(%d not in [1, %d])", b.GPITrigger.Port, d.nGPIs)
	}

	transmit, err := d.Transmit(b)
	if err != nil {
		return nil, err
	}

	_, best := d.findBestMode(e.NumNearbyReaders)

	// Impinj doesn't support state aware filtering via standard LLRP messages,
	// but does support the concept via a custom parameter they call "Search modes".
	queryAction := &C1G2SingulationControl{}
	searchMode := impjDualTarget

	switch b.ScanType {
	case ScanFast:
		queryAction = &C1G2SingulationControl{
			Session:        0,
			TagPopulation:  500,
			TagTransitTime: 500,
		}

		if b.ImpinjOptions != nil && b.ImpinjOptions.SuppressMonza {
			queryAction.Session = 1 // TagFocus only makes sense with S1
			searchMode = impjSingleTargetSuppressed
		}
	case ScanNormal:
		queryAction = &C1G2SingulationControl{
			Session:        1,
			TagPopulation:  1000,
			TagTransitTime: 5000,
		}

		if b.ImpinjOptions != nil && b.ImpinjOptions.SuppressMonza {
			searchMode = impjSingleTargetSuppressed
		}
	case ScanDeep:
		searchMode = impjDualTargetWithReset
		queryAction = &C1G2SingulationControl{
			Session:        2,
			TagPopulation:  3000,
			TagTransitTime: 10000,
		}
	}

	return &ROSpec{
		ROSpecID:       1, // May be overridden, but better to ensure it's not 0.
		ROBoundarySpec: b.Boundary(),
		AISpecs: []AISpec{{
			AntennaIDs: []AntennaID{0},
			InventoryParameterSpecs: []InventoryParameterSpec{{
				InventoryParameterSpecID: 1,
				AirProtocolID:            AirProtoEPCGlobalClass1Gen2,
				AntennaConfigurations: []AntennaConfiguration{{
					AntennaID:     0,
					RFTransmitter: transmit,
					C1G2InventoryCommand: &C1G2InventoryCommand{
						RFControl: &C1G2RFControl{
							RFModeID: uint16(best.ModeID),
						},
						SingulationControl: queryAction,
						Custom: []Custom{{
							VendorID: uint32(PENImpinj),
							Subtype:  ImpinjSearchMode,
							Data:     []byte{uint8(searchMode >> 8), uint8(searchMode & 0xFF)},
						}},
					},
				}},
			}},
		}},
	}, nil
}

func (b Behavior) Boundary() ROBoundarySpec {
	return ROBoundarySpec{
		StartTrigger: b.StartTrigger(),
		StopTrigger:  b.stopTrigger(),
	}
}

// StartTrigger returns an llrp.ROSpecStartTrigger for the Behavior.
//
// If the Behavior includes a GPITrigger, the returned StartTrigger
// only starts the ROSpec if the GPITrigger conditions match.
// Otherwise, the returned StartTrigger is configured
// so that it'll start the ROSpec immediately once Enabled.
func (b Behavior) StartTrigger() (t ROSpecStartTrigger) {
	if b.GPITrigger == nil {
		if b.Duration == 0 {
			t.Trigger = ROStartTriggerImmediate
		} else {
			t.Trigger = ROStartTriggerNone
		}
	} else {
		t.Trigger = ROStartTriggerGPI
		copyTrigger := GPITriggerValue(*b.GPITrigger)
		t.GPITrigger = &copyTrigger
	}
	return
}

// stopTrigger returns an llrp.ROSpecStopTrigger for the Behavior.
//
// If the Behavior Duration is 0, this returns a StopTrigger
// that runs the ROSpec until the Reader explicitly receives a StopROSpec command.
// Otherwise, the returned StopTrigger is configured to stop the ROSpec
// after the Duration milliseconds.
func (b Behavior) stopTrigger() (t ROSpecStopTrigger) {
	if b.Duration > 0 {
		t.Trigger = ROStopTriggerDuration
		t.DurationTriggerValue = b.Duration
	}
	return
}

var (
	scanStrs = [...][]byte{
		ScanFast:   []byte("Fast"),
		ScanNormal: []byte("Normal"),
		ScanDeep:   []byte("Deep"),
	}
)

func (s ScanType) MarshalText() ([]byte, error) {
	if !(0 <= int(s) && int(s) < len(scanStrs)) {
		return nil, errors.Errorf("unknown ScanType: %v", s)
	}
	return scanStrs[s], nil
}

func (s *ScanType) UnmarshalText(text []byte) error {
	for i := range scanStrs {
		if bytes.Equal(scanStrs[i], text) {
			*s = ScanType(i)
			return nil
		}
	}

	return errors.Errorf("unknown ScanType: %q", string(text))
}
