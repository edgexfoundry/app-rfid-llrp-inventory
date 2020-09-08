//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package routes

import (
	"github.impcloud.net/RSP-Inventory-Suite/rfid-inventory/internal/llrp"
	"time"
)

const (
	reportInterval = uint16((time.Second * 5) / 1e6) // how often to send tag reports; must be <=65.535s
)

// newReaderConfig returns a new llrp.SetReaderConfig.
// The JSON marshaled version of this message can be sent directly
// to the paths the command service reports for SetReaderConfig.
//
// If you're trying to affect a Reader's behavior during singulation,
// most of your control falls under AntennaConfiguration,
// and specifically within that, the Inventory Command parameters,
// and specifically within *that*, you're really interested in the RFControl,
// which lets you pick an RFModeID from a Reader's UHFBandCapabilities.
func newReaderConfig() *llrp.SetReaderConfig {
	conf := &llrp.SetReaderConfig{
		ResetToFactoryDefaults: true, // Factory defaults depend on the Reader.

		GPOWriteData:         nil, // limited by GeneralDeviceCapabilities.GPIOCapabilities.NumGPOs
		GPIPortCurrentStates: nil, // limited by GeneralDeviceCapabilities.GPIOCapabilities.NumGPIs
		AccessReportSpec:     nil, // when set, 0 == within ROReport; 1 == immediately upon completion.

		// If EventsAndReports is true,
		// the Reader won't send us anything until we send EnableEventsAndReports.
		// It requires the LLRPCapabilities.SupportsEventsAndReportHolding.
		EventsAndReports: nil,

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

// newROSpec returns a new, default ROSpec instance with various assumptions.
// It should be adjusted based on desired behavior and device capabilities.
//
// Because the returned spec uses a StartTrigger set to Immediate,
// if you send EnableROSpec with this ROSpecID (default 1)
// the Reader should start reading immediately using its settings.
// You can stop it via DisableROSpec or DeleteROSpec.
// Enable, Disable, and Delete all permit an ROSpecID of 0 to target all ROSpecs.
//
// Although this is valid for any LLRP device, LLRP admits many restrictions.
// Most can be determined via the device's capabilities.
// Moreover, manufacturers may have restrictions
// that cannot be determined via standard LLRP messages.
// At present no particular manufacturer checks are imposed.
//
// Many of the values here are nil to use the Reader's Configuration defaults.
// Note that the ROReportSpec settings completely override the defaults,
// and they can have effects on the meaning of nil in TagDataReports.
//
// There can be at most LLRPCapabilities.MaxROSpecs,
// each with LLRPCapabilities.MaxSpecsPerROSpec.
// Impinj Speedways allows only a single ROSpec,
// but in general LLRP devices may support more than one,
// which could be useful for interesting behaviors.
func newROSpec() *llrp.ROSpec {
	return &llrp.ROSpec{
		ROSpecID:           1,                        // This must be >=1; used to enable/disable/start/stop/delete.
		Priority:           0,                        // 0==highest; Impinj requires 0. The limit is LLRPCapabilities.MaxPriorityLevelSupported or 7.
		ROSpecCurrentState: llrp.ROSpecStateDisabled, // ROSpecs must be Disabled upon creation.

		// Using a single ROSpec with StartTrigger=Immediate and StopTrigger=None,
		// we can use the Enable and Disable ROSpec messages as start/stop
		// without necessarily needing to add/delete the spec.
		//
		// If a Reader supports multiple ROSpecs, we could set StartTrigger to None.
		// Then it'd only start via StartROSpec for "one-off" behaviors.
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
}

// newReadTIDAccessSpec returns an llrp.AccessSpec for reading C1G2 TIDs.
// IMPORTANT: we are probably not ready to use this yet!
//
// The returned AccessSpec is disabled and uses ID=1 and Stop Trigger="None".
// It applies at all antennas, for all tags, and runs with all ROs.
// The internal Read OpSpec uses ID=1 and assumes no password is needed.
//
// The word count used is the expected TID bank size.
// LLRP & the EPC standard make it clear that WordCount can be 0,
// in which case a tag should backscatter the memory bank's full contents.
// Unfortunately, Impinj isn't compliant and will refuse such an OpSpec.
// Their docs specify they permit a max word count of 60 words.
//
// - There can be at most LLRPCapabilities.MaxAccessSpecs.
// - C1G2LLRPCapabilities limits the number of operations & filters per query.
// - MaxOpSpecsPerAccessSpec limits the number of operations per spec.
// - SupportsClientRequestOpSpec limits AccessSpec type.
// - If supported, ClientRequestedOpSpecTimeout limits its timeout.
//
// It's worth noting that Impinj has specific limits depending on the firmware,
// and most of these limitations cannot be determined via LLRP.
// For example, in 5.12, sending more than 9 BlockWrites
// will just silently ignore any past the 9th,
// and if the sum of WordCounts across all BlockWrites in an AccessSpec is >= 64
// or it results "in undefined behavior and may cause the Reader to crash"
// (Impinj Octane LLRP, v5.12.0, p23).
// The v6.4 manual no longer carries this warning,
// but both specify an individual Write or Block write must be <=32 words;
// Reads are limited to 60 words and don't support "0",
// which otherwise would (in the Gen2 protocol) read the full memory bank.
//
// To be clear: other manufacturers likely have their own limitations.
// These only mention Impinj because that information was readily available.
// Their docs are good reference, since they helped write the LLRP standard.
// The point is, to make proper use of these things for general LLRP devices,
// we must know the relevant limits for the specific manufacturer.
func newReadTIDAccessSpec(wordCount uint16) *llrp.AccessSpec {
	const TIDMemoryBank = 2

	return &llrp.AccessSpec{
		AccessSpecID:  1, // must be >= 1
		AirProtocolID: llrp.AirProtoEPCGlobalClass1Gen2,
		AccessCommand: llrp.AccessCommand{
			C1G2Read: &llrp.C1G2Read{
				OpSpecID:       1,
				C1G2MemoryBank: TIDMemoryBank,
				WordCount:      wordCount,
			},
		},
	}
}
