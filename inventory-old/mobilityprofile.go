/* Apache v2 license
*  Copyright (C) <2020> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	assetTrackingDefault = MobilityProfile{
		ID:            "asset_tracking_default",
		Slope:         -0.008,
		Threshold:     6.0,
		HoldoffMillis: 0.0,
	}

	retailGarmentDefault = MobilityProfile{
		ID:            "retail_garment_default",
		Slope:         -0.0005,
		Threshold:     6.0,
		HoldoffMillis: 60000.0,
	}

	defaultProfile = MobilityProfile{
		ID:            "default",
		Slope:         assetTrackingDefault.Slope,
		Threshold:     assetTrackingDefault.Threshold,
		HoldoffMillis: assetTrackingDefault.HoldoffMillis,
	}

	mobilityProfiles = map[string]MobilityProfile{
		assetTrackingDefault.ID: assetTrackingDefault,
		retailGarmentDefault.ID: retailGarmentDefault,
		defaultProfile.ID:       defaultProfile,
	}

	activeProfile = getDefaultMobilityProfile()
)

// Mobility Profile defines the parameters of the weighted slope formula used in calculating a tag's location.
// Tag location is determined based on the quality of tag reads associated with a sensor/antenna averaged over time.
// For a tag to move from one location to another, the other location must be either a better signal or be more recent.
type MobilityProfile struct {
	ID string `json:"id"`
	// Slope (dBm per millisecond): Used to determine the weight applied to older RSSI values
	Slope float64 `json:"m"`
	// Threshold (dBm) RSSI threshold that must be exceeded for the tag to move from the previous sensor
	Threshold float64 `json:"t"`
	// HoldoffMillis (milliseconds) Amount of time in which the weight used is just the threshold, effectively the slope is not used
	HoldoffMillis float64 `json:"a"`
	// b = y - (m*x)
	YIntercept float64 `json:"b"`
}

// b = y - (m*x)
func (profile *MobilityProfile) calculateYIntercept() {
	profile.YIntercept = profile.Threshold - (profile.Slope * profile.HoldoffMillis)
}

func GetActiveMobilityProfile() MobilityProfile {
	return activeProfile
}

func getDefaultMobilityProfile() MobilityProfile {
	profile, err := GetMobilityProfile(defaultProfile.ID)

	// default should always exist
	if err != nil {
		err = errors.Wrapf(err, "default mobility profile with id %s does not exist!", defaultProfile.ID)
		panic(err)
	}

	return profile
}

func GetMobilityProfile(id string) (MobilityProfile, error) {
	profile, ok := mobilityProfiles[id]
	if !ok {
		return MobilityProfile{}, fmt.Errorf("unable to find mobility profile with id: %s", id)
	}

	// check if y-intercept has been computed yet
	if profile.YIntercept == 0 {
		profile.calculateYIntercept()
		mobilityProfiles[profile.ID] = profile
	}

	return profile, nil
}
