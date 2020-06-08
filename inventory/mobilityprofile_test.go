/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import "testing"

func TestNewMobilityProfile(t *testing.T) {
	// check that default is asset tracking
	mp := getDefaultMobilityProfile()
	if mp.Slope >= 0.0 {
		t.Errorf("mobility profile: M is %v, which is >= 0.0.\n\t%#v", mp.Slope, mp)
	}
	if mp.Threshold != mp.YIntercept {
		t.Errorf("mobility profile: T of %v is NOT equal to B of %v, but they should be equal.\n\t%#v", mp.Threshold, mp.YIntercept, mp)
	}
}
