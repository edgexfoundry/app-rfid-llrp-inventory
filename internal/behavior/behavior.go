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
// The original API comes from the RSP Controller.
// LLRP Readers vary wildly in their capabilities;
// some Behavior characteristics cannot be well-mapped to all Readers.
type Behavior struct {
	ID string

	GPITrigger *GPITrigger `json:",omitempty"` // nil == Immediate; otherwise requires Port
	ScanType   ScanType
	Power      PowerTarget
	Duration   llrp.Millisecs32
}

type GPITrigger struct {
	Port    uint16
	Event   bool
	Timeout llrp.Millisecs32 `json:",omitempty"`
}

type PowerTarget struct {
	Min, Max *uint16
	Target   uint16
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

type SpecOption interface {
	ModifyROSpec(spec *llrp.ROSpec)
}

type SpecOptFunc func(spec *llrp.ROSpec)

func (sof SpecOptFunc) ModifyROSpec(spec *llrp.ROSpec) {
	sof(spec)
}

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

type DeviceInfo struct {
	modes       []llrp.UHFC1G2RFModeTableEntry
	pwrMinToMax []llrp.TransmitPowerLevelTableEntry
	nGPIs       uint16
	nFreqs      uint16
	freqIdx     uint16
	allowsHop   bool
	stateAware  bool
}

func NewDeviceInfo(c *llrp.GetReaderCapabilitiesResponse) (*DeviceInfo, error) {
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

	var freqIdx, nFreqs uint16
	freqInfo := regCap.UHFBandCapabilities.FrequencyInformation
	if freqInfo.Hopping {
		if len(freqInfo.FrequencyHopTables) == 0 {
			return nil, errMissingCapInfo("frequency table",
				"RegulatoryCapabilities", "UHFBandCapabilities",
				"FrequencyInformation", "FrequencyHopTables")
		}

		// LLRP's "abstract" definition of FrequencyHopTable says
		// "HopTableID: Integer; Possible Values: 0-255"
		// and the binary encoding allocates it 8-bits,
		// yet in the RFTransmitter parameter, it's defined as a uint16.
		freqIdx = uint16(freqInfo.FrequencyHopTables[0].HopTableID)
		// Array fields in LLRP are limited to at most a length of uint16
		// due to the way the binary values are encoded,
		// so if it's greater than a uint16,
		// the value didn't actually come from an LLRP message.
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
		if len(freqInfo.FrequencyHopTables[0].Frequencies) > (1 << 16) {
			panic("impossible frequency table length")
		}
		freqIdx = 1
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

	return &DeviceInfo{
		modes:       modes,
		pwrMinToMax: pwrLvls,
		nGPIs:       genCap.GPIOCapabilities.NumGPIs,
		nFreqs:      nFreqs,
		freqIdx:     freqIdx,
		allowsHop:   freqInfo.Hopping,
		stateAware:  llrpCap.CanDoTagInventoryStateAwareSingulation,
	}, nil
}

func (d DeviceInfo) freqIndices() (hopIdx, fixedIdx uint16) {
	if d.allowsHop {
		return d.freqIdx, 0
	}
	return 0, d.freqIdx
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
func (d DeviceInfo) findPower(target dBmX10) (tableIdx uint16, value dBmX10) {
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

func (d DeviceInfo) Fast(nReaders uint) (bestIdx int, mode llrp.UHFC1G2RFModeTableEntry) {
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
// which is validated when the DeviceInfo is created.
//
// LLRP (and this code) abstracts reader density via the mode's "SpectralMask"
// (the name relates to how Readers make use of the available channel spectrum).
// A higher mask level implies a more dense Reader environment:
// one in which most or all available frequency channels are occupied.
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
func (d DeviceInfo) fastestAt(mask llrp.SpectralMaskType) (bestIdx int, ok bool) {
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
	// it's perhaps possible to optimize Tari over time
	// by sampling the 0:1 ratio of tags in the antenna's FoV;
	// however, many manufacturers set Min Tari == Max Tari,
	// effectively forcing only a single value anyway.

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
func (d DeviceInfo) InventoryControl(b Behavior) ([]llrp.C1G2Filter, *llrp.C1G2SingulationControl) {
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
		TruncateAction:      llrp.FilterActionDoNotTruncate,
		TagInventoryMask:    llrp.C1G2TagInventoryMask{},
		AwareFilterAction:   selectAction,
		UnawareFilterAction: nil,
	}}, queryAction
}

func (b Behavior) NewROSpec(d DeviceInfo) (*llrp.ROSpec, error) {
	if b.GPITrigger != nil && (b.GPITrigger.Port == 0 ||
		d.nGPIs == 0 || b.GPITrigger.Port > d.nGPIs) {
		return nil, errors.Wrapf(ErrUnsatisfiable,
			"behavior uses a GPI Trigger with invalid Port "+
				"(%d not in [1, %d])", b.GPITrigger.Port, d.nGPIs)
	}

	pwrIdx, pwr := d.findPower(b.Power.Target)
	if pwr > b.Power.Target {
		return nil, errors.Wrapf(ErrUnsatisfiable,
			"target power (%fdBm) exceeds lowest supported (%fdBm)",
			float32(b.Power.Target)/10.0, float32(pwr)/10.0)
	}

	hopIdx, fixedIdx := d.freqIndices()

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
					AntennaID: 0,
					RFTransmitter: &llrp.RFTransmitter{
						TransmitPowerIndex: pwrIdx,
						HopTableID:         hopIdx,
						ChannelIndex:       fixedIdx,
					},
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

type Metric func(i int, v uint16) (matches bool, dist float64)

func FindExact(target uint16, listLen int, m Metric) (int, bool) {
	for i := 0; i < listLen; i++ {
		if match, _ := m(i, target); match {
			return i, true
		}
	}
	return 0, false
}

func FindClose(target uint16, tolerance float64, listLen int, m Metric) (bestIdx int, found bool) {
	for i := 0; i < listLen; i++ {
		match, d := m(i, target)
		if match {
			return i, true
		}

		if d < tolerance {
			found = true
			bestIdx = i
			tolerance = d
		}
	}

	return
}

func L1Distance(p []llrp.TransmitPowerLevelTableEntry) Metric {
	return func(i int, v uint16) (matches bool, dist float64) {
		if i < len(p) {
			pwr := p[i].TransmitPowerValue
			matches = pwr == v
			if pwr > v {
				dist = float64(pwr - v)
			} else {
				dist = float64(v - pwr)
			}
		}
		return
	}
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
