//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

// mobilityProfile defines the parameters of the weighted slope formula used in calculating a tag's location.
// Tag location is determined based on the quality of tag reads associated with a sensor/antenna averaged over time.
// For a tag to move from one location to another, the other location must be either a better signal or be more recent.
type mobilityProfile struct {
	// slope (dBm per millisecond): Used to determine the weight applied to older RSSI values
	slope float64
	// threshold (dBm) RSSI threshold that must be exceeded for the tag to move from the previous sensor
	threshold float64
	// holdoffMillis (milliseconds) Amount of time in which the weight used is just the threshold, effectively the slope is not used
	holdoffMillis float64
	// b = y - (m*x)
	yIntercept float64
}

func newMobilityProfile(slope, threshold, holdoffMillis float64) mobilityProfile {
	return mobilityProfile{
		slope:         slope,
		threshold:     threshold,
		holdoffMillis: holdoffMillis,
		yIntercept:    threshold - (slope * holdoffMillis), // b = y - (m*x)
	}
}

// computeOffset computes the offset to be applied to a value based on the time it was read vs the reference timestamp.
// Offsets can be positive or negative. Typically they will start out positive, and the longer the duration
// between the reference time and the lastRead, the more negative the offset will become.
func (profile *mobilityProfile) computeOffset(referenceTimestamp int64, lastRead int64) float64 {
	// y = mx + b
	offset := (profile.slope * float64(referenceTimestamp-lastRead)) + profile.yIntercept

	// check if offset needs to be capped at threshold ceiling
	if offset > profile.threshold {
		offset = profile.threshold
	}
	return offset
}
