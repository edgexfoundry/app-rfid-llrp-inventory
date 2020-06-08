/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

type rssiAdjuster struct {
	mobilityProfile MobilityProfile
}

func newRssiAdjuster() rssiAdjuster {
	return rssiAdjuster{
		mobilityProfile: GetActiveMobilityProfile(),
	}
}

func (adjuster *rssiAdjuster) getWeight(lastRead int64) float64 {
	profile := adjuster.mobilityProfile

	weight := (profile.Slope * float64(UnixMilliNow() - lastRead)) + profile.YIntercept

	// check if weight needs to be capped at threshold ceiling
	if weight > profile.Threshold {
		weight = profile.Threshold
	}

	return weight
}
