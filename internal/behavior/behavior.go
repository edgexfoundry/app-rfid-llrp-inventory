//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

/*
Package behavior implements higher-level logic atop an LLRP Reader.

This package converts <LLRP Reader Info, Desired Behavior> to LLRP messages & parameters.
*/
package behavior

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"sort"
	"strings"
)

// Behavior is a high-level description of desired Reader operation.
//
// LLRP Readers vary wildly in their capabilities;
// some Behavior characteristics cannot be well-mapped to all Readers.
type Behavior struct {
	ID string

	GPITrigger  *GPITrigger `json:",omitempty"` // nil == Immediate; otherwise requires Port
	ScanType    ScanType
	Duration    llrp.Millisecs32 // 0 == repeat forever
	Power       PowerTarget
	Frequencies []llrp.Kilohertz `json:",omitempty"`
}

type GPITrigger struct {
	Port    uint16
	Event   bool
	Timeout llrp.Millisecs32 `json:",omitempty"`
}

type PowerTarget struct {
	Target uint16
}

type (
	ScanType int
	Trigger  int
)

const (
	ScanFast            = ScanType(0)
	ScanFastSuppression = ScanType(1)
	ScanNormal          = ScanType(2)
	ScanDeep            = ScanType(3)

	TriggerImmediate = Trigger(0)
	TriggerGPI       = Trigger(1)
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

type BasicDevice struct {
	// connected   time.Time
	modes       []llrp.UHFC1G2RFModeTableEntry
	pwrMinToMax []llrp.TransmitPowerLevelTableEntry
	freqInfo    llrp.FrequencyInformation

	// report is the collection of information we want expect a Reader to report.
	// LLRP has a data compression "feature" that allows Readers to omit some parameters
	// if the value hasn't changed "since the last time it was sent".
	report llrp.TagReportContentSelector
	// lastData is the value of tag parameter the last time it was reported.
	lastData llrp.TagReportData

	nGPIs, nFreqs uint16
	allowsHop     bool
	stateAware    bool
}

func (d BasicDevice) NewReaderConfig() *llrp.SetReaderConfig {
	conf := &llrp.SetReaderConfig{
		ResetToFactoryDefaults: true,

		// ReaderEventNotificationSpec which ReaderEventNotifications we get.
		// BufferOverflows and ConnectionEvents cannot be disabled.
		// Some events require specific capabilities.
		ReaderEventNotificationSpec: &llrp.ReaderEventNotificationSpec{
			EventNotificationStates: []llrp.EventNotificationState{
				{ReaderEventType: llrp.NotifyReaderException, NotificationEnabled: true}, // notifies of unexpected Reader events
				{ReaderEventType: llrp.NotifyAntenna, NotificationEnabled: true},         // (dis)connect may require the Reader tries to use antenna
				// {ReaderEventType: llrp.NotifyROSpec, NotificationEnabled: false},                // ROSpec start/end/preempt
				// {ReaderEventType: llrp.NotifyAISpec, NotificationEnabled: false},                // AISpec end
				// {ReaderEventType: llrp.NotifyAISpecWithSingulation, NotificationEnabled: false}, // AISpec end & has singulation details
				// {ReaderEventType: llrp.NotifyReportBuffFillWarn, NotificationEnabled: true},    // requires LLRPCapabilities.CanReportBufferFillWarning
				// {ReaderEventType: llrp.NotifyRFSurvey, NotificationEnabled: false},             // requires LLRPCapabilities.CanDoRFSurvey
				// {ReaderEventType: llrp.NotifyChannelHop, NotificationEnabled: false},           // only relevant if RegulatoryCapabilities.UHFBandCapabilities.FrequencyInformation.Hopping
				// {ReaderEventType: llrp.NotifyGPI, NotificationEnabled: true},                   // only relevant if GeneralDeviceCapabilities.GPIOCapabilities has >0 NumGPIs or NumGPOs
			},
		},

		// Setting this requires GeneralDeviceCapabilities.CanSetAntennaProperties,
		// but it's not really that useful anyway.
		AntennaProperties: nil,

		// AntennaConfigurations control the RF settings during a Reader Operation.
		// The valid values are heavily limited by capabilities and manufacturer.
		// We can nil here and in ROSpecs to "let the Reader decide".
		//
		// Most of the inventory operation control comes via the RFControl.
		// RFControl is where you can choose a Mode from the
		// RegulatoryCapabilities.UHFBandCapabilities.C1G2RFModes table.
		// You match one of the RFModeIDs (which are not sequential, in general)
		// and give an in-range Tari value or 0 for "let the Reader choose".
		//
		// Using an AntennaID of 0 means the config applies to all antennas.
		// Antennas IDs need not be consecutive, but in practice, they probably are.
		// The max is limited by GeneralDeviceCapabilities.MaxSupportedAntennas.
		// Although we can query the Reader Config for Antenna Properties,
		// it can take much longer than typical LLRP requests
		// (e.g., Impinj quotes that it can take up to 10s to probe the ports).
		// Instead, we can determine the IDs from the GeneralDeviceCapabilities
		// within its PerAntennaAirProtocols parameter.
		//
		// If set, the RFReceiver must match one of the Index values
		// of one of the entries of the GeneralDeviceCapabilities.ReceiveSensitivityTable.
		// Those ReceiveSensitivity values are relative the Reader's max,
		// which cannot be determined from standard messages in LLRP 1.0.1.
		// Note that the Index values are not sequential, in general.
		//
		// If you're setting the RFTransmitter values, you must use
		// the RegulatoryCapabilities.UHFBandCapabilities
		// to determine a valid Index into the TransmitPowerLevels table
		// and whether or not the Reader is in a Frequency Hopping regulatory region.
		// If so, then HopTableID must match an entry in the FrequencyHopTables.
		// If not, ChannelIndex must be set based on a desired value
		// from the FixedFrequencyTable, which unlike most LLRP tables,
		// is a regular array, so the value is the 1-indexed offset.
		//
		// If LLRPCapabilities.CanDoTagInventoryStateAwareSingulation,
		// you can't use InventoryStateAwareActions in Filters or SingulationControl.
		//
		// The number of filters is limited by C1G2LLRPCapabilities.MaxSelectFilterPerQuery.
		// Which filter actions are available depends on
		// LLRPCapabilities.CanDoTagInventoryStateAwareSingulation.
		//
		// Note that even though LLRP requires compliant Readers implement the TruncateAction,
		// Impinj isn't compliant and their documentation says "Truncate must be 0".
		// LLRP allows per-antenna configurations,
		// but at least some Readers will reject variant per-antenna configs.
		// For example, the Impinj Speedway requires all antenna
		// in an enabled AISpec have the same RFControl.ModeIndex,
		// RFTransmitter.HopTableID and ChannelIndex, and C1G2Filters.
		// There's no way to know this via standard LLRP messages.
		//
		// Note that the Speedway has only a single HopTable,
		// and it doesn't support state aware filtering,
		// nor does it support setting the Tari value (as modes' range min == max),
		// nor is it compliant with LLRP w.r.t. TruncateActions.
		//
		// So here's what can be controlled per-antenna on a Speedway:
		// - The transmit power
		// - The receive sensitivity
		//
		// Here's what can be configured, if they match for all antennas:
		// - Up to 2-5 C1G2 filters, depending on the firmware,
		//   but only the mask & state-unaware filter action, but not truncation.
		// - The Singulation control's session flag;
		//   other Singulation controls have little effect or are ignored.
		// - 1 of 5 C1G2 RFMode (of the 540,000,000,000,000 possible)
		//   or you can use 1 of 5 Impinj-specific modes masquerading as LLRP RFModes.
		//   They call them "Autoset modes", and Impinj says of them,
		//   "Link Parameters reported for Autoset modes [...] should be ignored",
		//   so there's no way to use LLRP to determine their effect
		//   nor to disambiguate them from actual LLRP modes, for that matter.
		//   This is not to say that they aren't potentially useful,
		//   but it is to say they don't fit well with a general-purpose LLRP Device Service.
		AntennaConfigurations: nil,

		// The ROReportSpec controls default reporting parameters.
		// If an ROSpec has a non-nil ReportSpec, none of these apply.
		// The report settings (here or in the ROSpec if it overrides it)
		// must be known to in order to disambiguate nil data in a TagDataReport.
		ROReportSpec: &llrp.ROReportSpec{
			Trigger: llrp.NSecondsOrAIEnd,
			N:       reportInterval,

			// The ContentSelector controls what's eligible to come in a TagDataReport.
			//
			// If a report bundles multiple tag singulations together,
			// the report can include when it was first and/or last seen,
			// the total number of times the Reader saw it,
			// and its peak RSSI of all times it saw it.
			// Which values to enable in the report depends on the Trigger type,
			// since that determines whether it's even possible
			// that a single tag is seen multiple times.
			TagReportContentSelector: llrp.TagReportContentSelector{
				EnableROSpecID:             false, // should set true if have >1 ROSpec
				EnableSpecIndex:            false, // should set true if >1 Spec in ROSpec
				EnableInventoryParamSpecID: false, // should set true if have >1 within any AISpec
				EnableAccessSpecID:         false, // should set true if >1 AccessSpec
				EnableTagSeenCount:         false, // maybe want this to be true if bundling is possible
				EnableFirstSeenTimestamp:   false,
				EnableLastSeenTimestamp:    true,
				EnableChannelIndex:         true, // channel index depends on the frequency table in use
				EnableAntennaID:            true,
				EnablePeakRSSI:             true,
				C1G2EPCMemorySelector: &llrp.C1G2EPCMemorySelector{
					CRCEnabled:     false,
					PCBitsEnabled:  false, // PCBits can help distinguish actual EPCs from custom tags
					XPCBitsEnabled: false, // requires C1G2LLRPCapabilities.SupportsXPC
				},
			},
		},
	}

	return conf
}

// FillAmbiguousNil handles the worst feature of LLRP: ambiguous nil parameters.
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
func (d BasicDevice) FillAmbiguousNil(tags []llrp.TagReportData) {
	for i := range tags {
		tag := &tags[i]
		if d.report.EnableROSpecID {
			if tag.ROSpecID == nil {
				tag.ROSpecID = new(llrp.ROSpecID)
				*tag.ROSpecID = *d.lastData.ROSpecID
			} else {
				*d.lastData.ROSpecID = *tag.ROSpecID
			}
		}

		if d.report.EnableSpecIndex {
			if tag.SpecIndex == nil {
				tag.SpecIndex = new(llrp.SpecIndex)
				*tag.SpecIndex = *d.lastData.SpecIndex
			} else {
				*d.lastData.SpecIndex = *tag.SpecIndex
			}
		}

		if d.report.EnableInventoryParamSpecID {
			if tag.InventoryParameterSpecID == nil {
				tag.InventoryParameterSpecID = new(llrp.InventoryParameterSpecID)
				*tag.InventoryParameterSpecID = *d.lastData.InventoryParameterSpecID
			} else {
				*d.lastData.InventoryParameterSpecID = *tag.InventoryParameterSpecID
			}
		}

		if d.report.EnableAntennaID {
			if tag.AntennaID == nil {
				tag.AntennaID = new(llrp.AntennaID)
				*tag.AntennaID = *d.lastData.AntennaID
			} else {
				*d.lastData.AntennaID = *tag.AntennaID
			}
		}

		if d.report.EnablePeakRSSI {
			if tag.PeakRSSI == nil {
				tag.PeakRSSI = new(llrp.PeakRSSI)
				*tag.PeakRSSI = *d.lastData.PeakRSSI
			} else {
				*d.lastData.PeakRSSI = *tag.PeakRSSI
			}
		}

		if d.report.EnableChannelIndex {
			if tag.ChannelIndex == nil {
				tag.ChannelIndex = new(llrp.ChannelIndex)
				*tag.ChannelIndex = *d.lastData.ChannelIndex
			} else {
				*d.lastData.ChannelIndex = *tag.ChannelIndex
			}
		}

		if d.report.EnableFirstSeenTimestamp {
			if tag.FirstSeenUTC == nil {
				tag.FirstSeenUTC = new(llrp.FirstSeenUTC)
				*tag.FirstSeenUTC = *d.lastData.FirstSeenUTC
			} else {
				*d.lastData.FirstSeenUTC = *tag.FirstSeenUTC
			}
		}

		if d.report.EnableLastSeenTimestamp {
			if tag.LastSeenUTC == nil {
				tag.LastSeenUTC = new(llrp.LastSeenUTC)
				*tag.LastSeenUTC = *d.lastData.LastSeenUTC
			} else {
				*d.lastData.LastSeenUTC = *tag.LastSeenUTC
			}
		}

		if d.report.EnableTagSeenCount {
			if tag.TagSeenCount == nil {
				tag.TagSeenCount = new(llrp.TagSeenCount)
				*tag.TagSeenCount = *d.lastData.TagSeenCount
			} else {
				*d.lastData.TagSeenCount = *tag.TagSeenCount
			}
		}
	}
}

func NewBasicDevice(c *llrp.GetReaderCapabilitiesResponse) (*BasicDevice, error) {
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
	// c1g2Cap := c.C1G2LLRPCapabilities

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
	pwrLvls := make([]llrp.TransmitPowerLevelTableEntry, len(tpl))
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
		modes:       modes,
		pwrMinToMax: pwrLvls,
		nGPIs:       genCap.GPIOCapabilities.NumGPIs,
		freqInfo:    freqInfo,
		nFreqs:      nFreqs,
		allowsHop:   freqInfo.Hopping,
		stateAware:  llrpCap.CanDoTagInventoryStateAwareSingulation,
		lastData: llrp.TagReportData{
			ROSpecID:                 new(llrp.ROSpecID),
			SpecIndex:                new(llrp.SpecIndex),
			InventoryParameterSpecID: new(llrp.InventoryParameterSpecID),
			AntennaID:                new(llrp.AntennaID),
			PeakRSSI:                 new(llrp.PeakRSSI),
			ChannelIndex:             new(llrp.ChannelIndex),
			FirstSeenUTC:             new(llrp.FirstSeenUTC),
			LastSeenUTC:              new(llrp.LastSeenUTC),
			TagSeenCount:             new(llrp.TagSeenCount),
		},
	}, nil
}

// transmit returns a legal llrp.RFTransmitter value.
func (d BasicDevice) transmit(b Behavior) (*llrp.RFTransmitter, error) {
	pwrIdx, pwr := d.findPower(b.Power.Target)
	if pwr > b.Power.Target {
		return nil, errors.Wrapf(ErrUnsatisfiable,
			"target power (%fdBm) exceeds lowest supported (%fdBm)",
			float32(b.Power.Target)/10.0, float32(pwr)/10.0)
	}

	if d.allowsHop {
		return &llrp.RFTransmitter{
			HopTableID:         uint16(d.freqInfo.FrequencyHopTables[0].HopTableID),
			TransmitPowerIndex: pwrIdx,
		}, nil
	}

	for _, wanted := range b.Frequencies {
		for i, f := range d.freqInfo.FixedFrequencyTable.Frequencies {
			if wanted == f {
				return &llrp.RFTransmitter{
					ChannelIndex: uint16(i),
				}, nil
			}
		}
	}

	return nil, errors.Wrapf(ErrUnsatisfiable, "no matching frequency available")
}

type dBmX10 = llrp.MillibelMilliwatt

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
func (d BasicDevice) findPower(target dBmX10) (tableIdx uint16, value dBmX10) {
	// sort.Search returns the smallest index i at which f(i) is true,
	// or the list len if the result is always false.
	// This requires the list is sorted (in our case, in ascending order).
	pwrIdx := sort.Search(len(d.pwrMinToMax), func(i int) bool {
		return d.pwrMinToMax[i].TransmitPowerValue >= target
	})

	var t llrp.TransmitPowerLevelTableEntry
	if pwrIdx == 0 {
		t = d.pwrMinToMax[pwrIdx]
	} else {
		t = d.pwrMinToMax[pwrIdx-1]
	}

	return t.Index, t.TransmitPowerValue
}

func (d BasicDevice) Fast(nReaders uint) (bestIdx int, mode llrp.UHFC1G2RFModeTableEntry) {
	const dense = 0.5 // EPC spec implies >50% is about where "multi" becomes "dense"
	var maskTarget llrp.SpectralMaskType
	switch nReaders {
	case 0:
		maskTarget = llrp.SpectralMaskUnknown
	case 1:
		maskTarget = llrp.SpectralMaskSingleInterrogator
	default:
		density := float64(nReaders) / float64(d.nFreqs)
		if nReaders >= uint(d.nFreqs) || density > dense {
			maskTarget = llrp.SpectralMaskDenseInterrogator
		} else {
			maskTarget = llrp.SpectralMaskMultiInterrogator
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

// fastestAt returns the index fastest of the UHFMode
// with a spectral mask at least as high as the input,
// where "fastest" is defined as described below;
// if there are no modes at or above the given density mask,
// the returned "ok" value is false.
//
// If the input mask level is 0 ("Unknown"),
// "ok" will be true and bestIdx will be valid.
//
// The returned bestIdx is the 0-indexed Go slice index,
// not the LLRP-defined ModeID of the relevant mode.
// There must be at least one mode in the mode table,
// which is validated when the BasicDevice is created.
//
// LLRP (and this code) abstracts reader density via the mode's "SpectralMask"
// (the name relates to how Readers make use of the available channel spectrum).
// A higher mask level implies a more dense Reader environment:
// one in which most or all available frequency freqInfo are occupied.
// Minimizing collisions requires frequency-division multiplexing,
// preferably by choosing backscattered link frequencies and modulations
// that permit guardbands between the carrier waves and backscattered sidebands.
// More information can be found in the
// EPC Radio-Frequency Identity Protocols Generation-2 UHF RFID Standard,
// particularly Appendix G.
//
// This code selects the "fastest" mode
// by first filtering out modes that expect lower density than the input,
// then sorts by the following categories, breaking ties by moving down the list:
//
// - higher BackscatterDataRate
// - lower PIERatio * MinTariTime
// - EPC HAG conformant over not conformant (unlikely in practice)
// - DR 8:1 over 64:3, since at the same BLF, it implies a smaller TRcal.
func (d BasicDevice) fastestAt(mask llrp.SpectralMaskType) (bestIdx int, ok bool) {
	// BDR = BLF at FM0, or BLF over 2, 4, or 8 for Miller modes,
	// so it already incorporates TRcal (Tpri = 1/BLF = TRcal / DR)
	// and subcarrier cycles per symbols (FM0 or Miller mode).
	// R->T calibration length is the sum of symbol lengths (RTcal = ZERO + ONE);
	// Tari gives us the data-0 length; it and PIERatio determines data-1.
	//
	// In theory, if you knew the symbol distribution (ratio of 0s to 1s),
	// you might could optimize the forward link by preferring
	// a shorter Tari with larger PIE when there are "enough" more 0s than 1s
	// and longer Tari with a smaller PIE if 1s outnumber 0s "enough".
	// Since that requires knowing the data distribution
	// (and only gives any benefit if the distribution is imbalanced),
	// the exact value of "enough" is left as an exercise for the open-source enthusiast.
	//
	// If you're particularly motivated and have a Reader that allows it,
	// it's perhaps possible to optimize Tari
	// by approximating the 0:1 ratio of messages the Reader will send;
	// however, many manufacturers set Min Tari == Max Tari,
	// effectively forcing only a single value anyway,
	// making such an optimization pretty pointless.

	var maxBwdRate llrp.BitsPerSec // higher is better
	var minFwdTime float64         // lower is better

	// unlikely further tie breakers
	var hagConf bool
	var divRatio llrp.DivideRatio

	for i, m := range d.modes {
		fwd := float64(m.MinTariTime) * float64(1000+m.PIERatio)
		// note that switch cases are evaluated in definition order;
		switch {
		case m.SpectralMask < mask:
			continue
		case m.BackscatterDataRate < maxBwdRate:
			continue
		case m.BackscatterDataRate > maxBwdRate: // BDR is better
		case fwd > minFwdTime: // break ties with Fwd link rate
			continue
		case fwd < minFwdTime: // Fwd link rate is better
		case !hagConf && m.IsEPCHagConformant: // see if one happens to be EPC HAG conformant
		case divRatio == llrp.DRSixtyFourToThree && m.DivideRatio == llrp.DREightToOne:
		default:
			continue // m is neither better nor worse
		}

		bestIdx = i
		minFwdTime = fwd
		maxBwdRate = m.BackscatterDataRate
		hagConf = m.IsEPCHagConformant
		divRatio = m.DivideRatio
	}
	return
}

// InventoryControl returns the C1G2 LLRP parameters that control how a Reader
// manages a tag population during Select/Inventory.
//
// A Reader singulates tags via a 3 step process: Select, Inventory, Access.
// LLRP abstracts the Select and Inventory parts llrp.C1G2InventoryCommand parameter,
// made up of an RF mode and the parameters returned by this method
// (the Access step is controlled via llrp.AccessSpec).
//
// The llrp.C1G2Filter parameters correspond to Select commands,
// while the llrp.C1G2SingulationControl parameter is for the Inventory stage.
// During Select, the Reader issues commands to force tags into certain states,
// then during Inventory, it tells tags in a certain states to respond,
// and maybe to change state if they're acknowledged by the Reader.
//
// More specifically, tags have 4 "session flags" and a "select" (SL) flag
// (there are some other flags, but they're newer than LLRP).
// During Select, the Reader sets a tag's session flags to A or B
// and its SL flag to "asserted" or "deasserted".
// If a tag goes unenergized long enough, its flags reset to A and deasserted.
// but eventually reset to A on their own, depending on the specific flag:
// - S0 holds its state while energized and resets if unpowered
// - S1 resets after a short period of time, regardless of power
// - S2 and S3 work the same: they reset if left unpowered long enough
// - SL starts deasserted and holds its state if unpowered long enough
//
// The EPC standard presents a basic example of how a Reader might use this:
// it can transition all tags in S2 from A to B,
// then once no more respond to an "S2+A" query,
// start transitioning the tags from B to A (still in S2).
// Meanwhile, another Reader could do the same, but in S3,
// and even if some of their tag populations overlapped,
// they won't "fight" over the tag states.
//
//
// Only Readers that have the CanDoTagInventoryStateAwareSingulation Capability
// permit the use of the llrp.C1G2TagInventoryStateAwareFilterAction
// and llrp.C1G2TagInventoryStateAwareSingulationAction,
// but this just means the Reader doesn't permit the Client to control it.
// The Reader still makes use of these commands,
// but the Client is "unaware" of the specific flags and actions.
// Using Filters with "Unaware" actions allows a Client to just request things like
// "Only give me tags that start with these EPC bits",
// and the Reader handles all that session flag stuff.
//
func (d BasicDevice) InventoryControl(b Behavior) ([]llrp.C1G2Filter, *llrp.C1G2SingulationControl) {
	// Flip session state for matching tags: A->B, B->A; do nothing non-matching tags
	const flipMatched = llrp.C1G2TagInventoryStateAwareFilterActionType(3)

	// Set matching tags session state to B; do nothing non-matching tags
	const setMatchB = llrp.C1G2TagInventoryStateAwareFilterActionType(5)

	var selectAction *llrp.C1G2TagInventoryStateAwareFilterAction
	queryAction := &llrp.C1G2SingulationControl{
		InvAwareAction: new(llrp.C1G2TagInventoryStateAwareSingulationAction),
	}

	switch b.ScanType {
	case ScanFast:
		// During Select, flip S0 (A->B, B->A).
		// S0 maintains state unless de-energized, at which point it resets to A.
		// During Query, ACK'd tags set S0 to A and SL to false.
		selectAction = &llrp.C1G2TagInventoryStateAwareFilterAction{
			Target:       llrp.InvTargetInventoriedS0,
			FilterAction: flipMatched,
		}
		queryAction = &llrp.C1G2SingulationControl{
			Session:        0,
			TagPopulation:  500,
			TagTransitTime: 500,
			InvAwareAction: &llrp.C1G2TagInventoryStateAwareSingulationAction{
				InventoryState: llrp.SingActAwareStateA,
				Selected:       false,
			},
		}
	case ScanFastSuppression:
		// During Select, set S0 to B
		// S1 maintains state for 0.5-5.0s, at which point it resets to A.
		selectAction = &llrp.C1G2TagInventoryStateAwareFilterAction{
			Target:       llrp.InvTargetInventoriedS0,
			FilterAction: setMatchB,
		}
		queryAction = &llrp.C1G2SingulationControl{
			Session:        0,
			TagPopulation:  500,
			TagTransitTime: 500,
			InvAwareAction: &llrp.C1G2TagInventoryStateAwareSingulationAction{
				InventoryState: llrp.SingActAwareStateA,
				Selected:       false,
			},
		}
	case ScanNormal:
		// During Selection, flip the S1 state of matched tags.
		// When tags enter the antenna FoV, their S1 flag is A,
		// so they'll get set to B and ignored during the round.
		// After 0.5-5.0s, their S1 flag will decay back to A.
		// S1 maintains state for 0.5-5.0s, at which point it resets to A.
		selectAction = &llrp.C1G2TagInventoryStateAwareFilterAction{
			Target:       llrp.InvTargetInventoriedS1,
			FilterAction: flipMatched,
		}
		queryAction = &llrp.C1G2SingulationControl{
			Session:        1,
			TagPopulation:  1000,
			TagTransitTime: 5000,
			InvAwareAction: &llrp.C1G2TagInventoryStateAwareSingulationAction{
				InventoryState: llrp.SingActAwareStateA,
				Selected:       false,
			},
		}
	case ScanDeep:
		// S2 maintains state while energized, but resets to A if de-energized >2s.
		selectAction = &llrp.C1G2TagInventoryStateAwareFilterAction{
			Target:       llrp.InvTargetInventoriedS2,
			FilterAction: setMatchB,
		}
		queryAction = &llrp.C1G2SingulationControl{
			Session:        2,
			TagPopulation:  3000,
			TagTransitTime: 10000,
			InvAwareAction: &llrp.C1G2TagInventoryStateAwareSingulationAction{
				InventoryState: llrp.SingActAwareStateA,
				Selected:       false,
			},
		}
	}

	return []llrp.C1G2Filter{{
		TruncateAction: llrp.FilterActionDoNotTruncate,
		TagInventoryMask: llrp.C1G2TagInventoryMask{
			MemoryBank: 1,
		},
		AwareFilterAction:   selectAction,
		UnawareFilterAction: nil,
	}}, queryAction
}

// NewROSpec returns a new llrp.ROSpec that implements the Behavior.
func (d BasicDevice) NewROSpec(b Behavior) (*llrp.ROSpec, error) {
	if b.GPITrigger != nil && (b.GPITrigger.Port == 0 ||
		d.nGPIs == 0 || b.GPITrigger.Port > d.nGPIs) {
		return nil, errors.Wrapf(ErrUnsatisfiable,
			"behavior uses a GPI Trigger with invalid Port "+
				"(%d not in [1, %d])", b.GPITrigger.Port, d.nGPIs)
	}

	transmit, err := d.transmit(b)
	if err != nil {
		return nil, err
	}

	mIdx, best := d.Fast(1)
	tari := d.modes[mIdx].MinTariTime
	filters, singulation := d.InventoryControl(b)

	spec := &llrp.ROSpec{
		ROBoundarySpec: b.boundary(),
		AISpecs: []llrp.AISpec{{
			AntennaIDs: []llrp.AntennaID{0}, // All
			InventoryParameterSpecs: []llrp.InventoryParameterSpec{{
				InventoryParameterSpecID: 1,
				AirProtocolID:            llrp.AirProtoEPCGlobalClass1Gen2,
				AntennaConfigurations: []llrp.AntennaConfiguration{{
					AntennaID:     0,
					RFTransmitter: transmit,
					C1G2InventoryCommand: &llrp.C1G2InventoryCommand{
						TagInventoryStateAware: false,
						RFControl: &llrp.C1G2RFControl{
							RFModeID: uint16(best.ModeID),
							Tari:     uint16(tari),
						},
						SingulationControl: singulation,
						Filters:            filters,
						Custom:             nil,
					},
				}},
			}},
		}},
	}
	return spec, nil
}

func (b Behavior) boundary() llrp.ROBoundarySpec {
	return llrp.ROBoundarySpec{
		StartTrigger: b.startTrigger(),
		StopTrigger:  b.stopTrigger(),
	}
}

func (b Behavior) startTrigger() (t llrp.ROSpecStartTrigger) {
	if b.GPITrigger == nil {
		t.Trigger = llrp.ROStartTriggerImmediate
	} else {
		t.Trigger = llrp.ROStartTriggerGPI
		t.GPITrigger = (*llrp.GPITriggerValue)(b.GPITrigger)
	}
	return
}

func (b Behavior) stopTrigger() (t llrp.ROSpecStopTrigger) {
	if b.Duration > 0 {
		t.Trigger = llrp.ROStopTriggerDuration
		t.DurationTriggerValue = b.Duration
	}
	return
}

var (
	scanStrs = [...][]byte{
		ScanFast:            []byte("Fast"),
		ScanFastSuppression: []byte("FastSuppression"),
		ScanNormal:          []byte("Normal"),
		ScanDeep:            []byte("Deep"),
	}

	triggerStrs = [...][]byte{
		TriggerImmediate: []byte("Immediate"),
		TriggerGPI:       []byte("GPI"),
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

func (t Trigger) MarshalText() ([]byte, error) {
	if !(0 <= int(t) && int(t) < len(triggerStrs)) {
		return nil, errors.Errorf("unknown Trigger: %v", t)
	}
	return triggerStrs[t], nil
}

func (t *Trigger) UnmarshalText(text []byte) error {
	for i := range triggerStrs {
		if bytes.Equal(triggerStrs[i], text) {
			*t = Trigger(i)
			return nil
		}
	}

	return errors.Errorf("unknown Trigger: %q", string(text))
}
