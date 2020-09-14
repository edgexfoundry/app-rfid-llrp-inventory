//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"reflect"
	"testing"
)

func TestDefaultAssetTracking(t *testing.T) {
	// check that default is asset tracking
	if !reflect.DeepEqual(mobilityProfiles["default"], mobilityProfiles["asset_tracking"]) {
		t.Errorf("default mobility profile does not match asset_tracking!")
	}
}

func TestNewMobilityProfile(t *testing.T) {
	// test sane values
	mp := loadMobilityProfile(lc)
	if mp.Slope >= 0.0 {
		t.Errorf("mobility profile: Slope is %v, but should be a negative number.\n\t%#v", mp.Slope, mp)
	}
	if mp.Threshold <= 0.0 {
		t.Errorf("mobility profile: Threshold is %v, but should be greater than 0.\n\t%#v", mp.Threshold, mp)
	}
	if mp.YIntercept != (mp.Threshold - (mp.Slope * mp.HoldoffMillis)) {
		t.Errorf("mobility profile: YIntercept of %v is NOT equal to expected: %v.\n\t%#v", mp.YIntercept, mp.Threshold-(mp.Slope*mp.HoldoffMillis), mp)
	}
}

func TestMobilityProfileOverrideThreshold(t *testing.T) {
	mp1 := loadMobilityProfile(lc)

	MobilityProfileThresholdOverridden = true
	MobilityProfileThreshold = mp1.Threshold * 2
	mp2 := loadMobilityProfile(lc)
	if mp2.Threshold != mp1.Threshold*2 {
		t.Errorf("mobility profile 2 threshold of %v does not equal the expected: %v", mp2.Threshold, mp1.Threshold*2)
	}
	if mp2.YIntercept == mp1.YIntercept {
		t.Errorf("mobility profile 2 should have a different Y-Iintercept than mobility profile 1!")
	}
	MobilityProfileThresholdOverridden = false
}

func TestMobilityProfileOverrideSlope(t *testing.T) {
	mp1 := loadMobilityProfile(lc)

	MobilityProfileSlopeOverridden = true
	MobilityProfileSlope = mp1.Slope * 2
	mp2 := loadMobilityProfile(lc)

	if mp2.Slope != mp1.Slope*2 {
		t.Errorf("mobility profile 2 Slope of %v does not equal the expected: %v", mp2.Slope, mp1.Slope*2)
	}
	if mp2.YIntercept == mp1.YIntercept {
		t.Errorf("mobility profile 2 should have a different Y-Iintercept than mobility profile 1!")
	}
	MobilityProfileSlopeOverridden = false
}

func TestMobilityProfileOverrideHoldoff(t *testing.T) {
	mp1 := loadMobilityProfile(lc)

	MobilityProfileHoldoffMillisOverridden = true
	MobilityProfileHoldoffMillis = mp1.HoldoffMillis + 10000
	mp2 := loadMobilityProfile(lc)

	if mp2.HoldoffMillis != mp1.HoldoffMillis+10000 {
		t.Errorf("mobility profile 2 HoldoffMillis of %v does not equal the expected: %v", mp2.HoldoffMillis, mp1.HoldoffMillis+10000)
	}
	if mp2.YIntercept == mp1.YIntercept {
		t.Errorf("mobility profile 2 should have a different Y-Iintercept than mobility profile 1!")
	}
	MobilityProfileHoldoffMillisOverridden = false
}
