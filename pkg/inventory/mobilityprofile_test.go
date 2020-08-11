/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"github.com/intel/rsp-sw-toolkit-im-suite-inventory-service/app/config"
	"reflect"
	"testing"
)

func TestDefaultAssetTracking(t *testing.T) {
	// check that default is asset tracking
	if ! reflect.DeepEqual(mobilityProfiles["default"], mobilityProfiles["asset_tracking"]) {
		t.Errorf("default mobility profile does not match asset_tracking!")
	}
}

func TestNewMobilityProfile(t *testing.T) {
	// test sane values
	mp := GetMobilityProfile()
	if mp.Slope >= 0.0 {
		t.Errorf("mobility profile: Slope is %v, but should be a negative number.\n\t%#v", mp.Slope, mp)
	}
	if mp.Threshold <= 0.0 {
		t.Errorf("mobility profile: Threshold is %v, but should be greater than 0.\n\t%#v", mp.Threshold, mp)
	}
	if mp.YIntercept != (mp.Threshold - (mp.Slope * mp.HoldoffMillis)) {
		t.Errorf("mobility profile: YIntercept of %v is NOT equal to expected: %v.\n\t%#v", mp.YIntercept, mp.Threshold - (mp.Slope * mp.HoldoffMillis), mp)
	}
}

func TestMobilityProfileOverrideThreshold(t *testing.T) {
	mp1 := loadMobilityProfile()

	config.AppConfig.MobilityProfileThresholdOverridden = true
	config.AppConfig.MobilityProfileThreshold = mp1.Threshold * 2
	mp2 := loadMobilityProfile()
	if mp2.Threshold != mp1.Threshold * 2 {
		t.Errorf("mobility profile 2 threshold of %v does not equal the expected: %v", mp2.Threshold, mp1.Threshold*2)
	}
	if mp2.YIntercept == mp1.YIntercept {
		t.Errorf("mobility profile 2 should have a different Y-Iintercept than mobility profile 1!")
	}
	config.AppConfig.MobilityProfileThresholdOverridden = false
}

func TestMobilityProfileOverrideSlope(t *testing.T) {
	mp1 := loadMobilityProfile()

	config.AppConfig.MobilityProfileSlopeOverridden = true
	config.AppConfig.MobilityProfileSlope = mp1.Slope * 2
	mp2 := loadMobilityProfile()

	if mp2.Slope != mp1.Slope * 2 {
		t.Errorf("mobility profile 2 Slope of %v does not equal the expected: %v", mp2.Slope, mp1.Slope*2)
	}
	if mp2.YIntercept == mp1.YIntercept {
		t.Errorf("mobility profile 2 should have a different Y-Iintercept than mobility profile 1!")
	}
	config.AppConfig.MobilityProfileSlopeOverridden = false
}

func TestMobilityProfileOverrideHoldoff(t *testing.T) {
	mp1 := loadMobilityProfile()

	config.AppConfig.MobilityProfileHoldoffMillisOverridden = true
	config.AppConfig.MobilityProfileHoldoffMillis = mp1.HoldoffMillis + 10000
	mp2 := loadMobilityProfile()

	if mp2.HoldoffMillis != mp1.HoldoffMillis + 10000 {
		t.Errorf("mobility profile 2 HoldoffMillis of %v does not equal the expected: %v", mp2.HoldoffMillis, mp1.HoldoffMillis + 10000)
	}
	if mp2.YIntercept == mp1.YIntercept {
		t.Errorf("mobility profile 2 should have a different Y-Iintercept than mobility profile 1!")
	}
	config.AppConfig.MobilityProfileHoldoffMillisOverridden = false
}
