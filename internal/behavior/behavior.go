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
)

// Behavior is a high-level description of desired Reader operation.
//
// The original API comes from the RSP Controller.
// LLRP Readers vary wildly in their capabilities;
// some Behavior characteristics cannot be well-mapped to all Readers.
type Behavior struct {
	ID       string
	ScanType ScanType

	Power PowerTarget

	Duration   llrp.Millisecs32 // 0 == Continuous
	GPITrigger *GPITrigger      `json:",omitempty"` // nil == Immediate
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

func WithSpecID(id llrp.ROSpecID) SpecOption {
	return SpecOptFunc(func(spec *llrp.ROSpec) {
		spec.ROSpecID = uint32(id)
	})
}

func WithMaxPriority(c *llrp.LLRPCapabilities) SpecOption {
	return SpecOptFunc(func(spec *llrp.ROSpec) {
		spec.Priority = c.MaxPriorityLevelSupported
	})
}

var (
	ErrMissingCapInfo = fmt.Errorf("missing capability information")
	ErrUnsatisfiable  = fmt.Errorf("behavior cannot be satisfied")
)

type missingInfo struct {
	path []string
}

func errMissingInfo(path ...string) *missingInfo {
	return &missingInfo{path: path}
}

func (b Behavior) NewROSpec(c *llrp.GetReaderCapabilitiesResponse) (*llrp.ROSpec, error) {
	if c == nil || c.LLRPCapabilities == nil || c.GeneralDeviceCapabilities == nil ||
		c.RegulatoryCapabilities == nil || c.C1G2LLRPCapabilities == nil {
		return nil, errors.New("missing capabilities")
	}

	genCap := c.GeneralDeviceCapabilities
	regCap := c.RegulatoryCapabilities
	llrpCap := c.LLRPCapabilities
	c1g2Cap := c.C1G2LLRPCapabilities

	if regCap == nil || regCap.UHFBandCapabilities == nil ||
		len(regCap.UHFBandCapabilities.TransmitPowerLevels) == 0 {
		return nil, errors.Wrap(ErrMissingCapInfo, "missing RegulatorCapabilities.UHFBandCapabilities.TransmitPowerLevels")
		("missing transmit power levels")
	}

	pls := regCap.UHFBandCapabilities.TransmitPowerLevels
	if b.Power.Min == nil && b.Power.Max == nil {
		i, ok := FindExact(b.Power.Target, len(pls), L1Distance(pls))
		if !ok {
			return nil, errors.Errorf("ca")
		}
	}

	spec := &llrp.ROSpec{}
	if err := b.setStartTrigger(genCap, spec); err != nil {
		return nil, err
	}

	b.setStopTrigger(spec)

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

func (b Behavior) setStartTrigger(genCap *llrp.GeneralDeviceCapabilities, spec *llrp.ROSpec) error {
	if b.GPITrigger == nil {
		spec.ROBoundarySpec.StartTrigger = llrp.ROSpecStartTrigger{
			Trigger:         llrp.ROStartTriggerImmediate,
			PeriodicTrigger: nil,
			GPITrigger:      nil,
		}
		return nil
	}

	if genCap.GPIOCapabilities.NumGPIs == 0 {
		return errors.Wrap(ErrBehaviorUnsatisfiable,
			"behavior uses a GPI Trigger, but Reader has no GPIs")
	}

	if b.GPITrigger.Port == 0 || b.GPITrigger.Port > genCap.GPIOCapabilities.NumGPIs {
		return errors.Wrap(ErrBehaviorUnsatisfiable,
			"behavior uses a GPI Trigger with an invalid Port")
	}

	t := llrp.GPITriggerValue(*b.GPITrigger)
	spec.ROBoundarySpec.StartTrigger = llrp.ROSpecStartTrigger{
		Trigger:         llrp.ROStartTriggerGPI,
		PeriodicTrigger: nil,
		GPITrigger:      &t,
	}

	return nil
}

func (b Behavior) setStopTrigger(spec *llrp.ROSpec) {
	if b.Duration == 0 {
		spec.ROBoundarySpec.StopTrigger = llrp.ROSpecStopTrigger{
			Trigger:              llrp.ROStopTriggerNone,
			DurationTriggerValue: 0,
			GPITriggerValue:      nil,
		}
	} else {
		spec.ROBoundarySpec.StopTrigger = llrp.ROSpecStopTrigger{
			Trigger:              llrp.ROStopTriggerDuration,
			DurationTriggerValue: 0,
			GPITriggerValue:      nil,
		}
	}
}

func NewROSpec(opts ...SpecOption) *llrp.ROSpec {
	s := &llrp.ROSpec{
		ROSpecID:           1,
		Priority:           0,                        // The limit is LLRPCapabilities.MaxPriorityLevelSupported or 7.
		ROSpecCurrentState: llrp.ROSpecStateDisabled, // ROSpecs must be Disabled upon creation.

		// Using a single ROSpec with StartTrigger=Immediate and StopTrigger=None,
		// we can use the Enable and Disable ROSpec messages as start/stop
		// without necessarily needing to add/delete the spec.
		ROBoundarySpec: llrp.ROBoundarySpec{
			StartTrigger: llrp.ROSpecStartTrigger{Trigger: llrp.ROStartTriggerImmediate},
			StopTrigger:  llrp.ROSpecStopTrigger{Trigger: llrp.ROStopTriggerNone},
		},

		// We must have at least one Spec. The max number is limited by capabilities.
		AISpecs: []llrp.AISpec{{
			StopTrigger: llrp.AISpecStopTrigger{Trigger: llrp.AIStopTriggerNone},

			// LLRP says given M AntennaIDs in this list
			// and N InventoryParameterSpecs in the following list,
			// the Reader will take M*N antenna inventory actions,
			// in whatever order the Reader prefers.
			// Additionally, it says a value of 0 targets all antennas.
			// It's not clear what happens given a list like [1, 2, 0, 2, 1],
			// and realistically, it's dependent on how the manufacturer did it.
			//
			// In any case, if you want to be specific,
			// then you're probably best off not using 0;
			// if you want to use 0 (all antennas),
			// you're probably best off only using 0.
			//
			// If we want a behavior that sequences antennas in a particular order,
			// then we need to generate separate AISpecs for each antenna.
			AntennaIDs: []llrp.AntennaID{0},
			InventoryParameterSpecs: []llrp.InventoryParameterSpec{{
				InventoryParameterSpecID: 1,   // must be >= 1
				AntennaConfigurations:    nil, // see notes in newReaderConfig
				AirProtocolID:            llrp.AirProtoEPCGlobalClass1Gen2,
			}},
		}},
		RFSurveySpecs: nil, // requires CanDoRFSurvey (not supported by Impinj Speedway)
		LoopSpec:      nil, // requires LLRP >=1.1
		ROReportSpec:  nil, // overrides all Reader Config settings
	}

	for _, opt := range opts {
		opt.ModifyROSpec(s)
	}

	return s
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
