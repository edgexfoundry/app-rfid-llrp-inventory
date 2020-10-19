package inventory

import (
	"math"
	"testing"
)

func TestNewMobilityProfile_yIntercept(t *testing.T) {
	tests := []struct {
		name                                        string
		slope, threshold, holdoffMillis, yIntercept float64
	}{
		{"asset_tracking", -0.008, 6, 500, 10},
		{"retail_garment", -0.0005, 6, 60000, 36},
		{"example_1", -0.1, 7, 350, 7 - (-0.1 * 350)},
		{"example_2", -0.049, 13, 1250, 13 - (-0.049 * 1250)},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mp := newMobilityProfile(test.slope, test.threshold, test.holdoffMillis)
			if math.Abs(mp.yIntercept-test.yIntercept) > epsilon {
				t.Errorf("Expected yIntercept to be: %v, but was: %v.", test.yIntercept, mp.yIntercept)
			}
		})
	}
}
