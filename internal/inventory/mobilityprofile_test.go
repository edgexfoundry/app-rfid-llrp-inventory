//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"testing"
)

func TestNewMobilityProfile(t *testing.T) {
	// test sane values
	cr := NewConfigurator(lc)
	consulConfig, err := cr.Parse(cr.defaultAppSettings)
	if err != nil {
		t.Fatalf("Error parsding default config: %v", err)
	}
	mp := loadMobilityProfile(consulConfig.ApplicationSettings)
	if mp.Slope >= 0.0 {
		t.Errorf("mobility profile: Slope is %v, but should be a negative number.\n\t%#v", mp.Slope, mp)
	}
	if mp.Threshold <= 0.0 {
		t.Errorf("mobility profile: Threshold is %v, but should be greater than 0.\n\t%#v", mp.Threshold, mp)
	}
	if mp.yIntercept != (mp.Threshold - (mp.Slope * mp.HoldoffMillis)) {
		t.Errorf("mobility profile: yIntercept of %v is NOT equal to expected: %v.\n\t%#v", mp.yIntercept, mp.Threshold-(mp.Slope*mp.HoldoffMillis), mp)
	}
}
