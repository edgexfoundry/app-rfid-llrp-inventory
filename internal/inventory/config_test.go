//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		Name                         string
		DepartedThresholdSeconds     uint
		DepartedCheckIntervalSeconds uint
		AgeOutHours                  uint
		ExpectError                  bool
	}{
		{
			Name:                         "Valid",
			DepartedThresholdSeconds:     600,
			DepartedCheckIntervalSeconds: 30,
			AgeOutHours:                  336,
			ExpectError:                  false,
		},
		{
			Name:                         "Invalid Departed Threshold",
			DepartedThresholdSeconds:     0,
			DepartedCheckIntervalSeconds: 30,
			AgeOutHours:                  336,
			ExpectError:                  true,
		}, {
			Name:                         "Invalid Departed Check Interval",
			DepartedThresholdSeconds:     600,
			DepartedCheckIntervalSeconds: 0,
			AgeOutHours:                  336,
			ExpectError:                  true,
		}, {
			Name:                         "Invalid Ageout",
			DepartedThresholdSeconds:     600,
			DepartedCheckIntervalSeconds: 30,
			AgeOutHours:                  0,
			ExpectError:                  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			appSettings := ApplicationSettings{
				DepartedThresholdSeconds:     tc.DepartedThresholdSeconds,
				DepartedCheckIntervalSeconds: tc.DepartedCheckIntervalSeconds,
				AgeOutHours:                  tc.AgeOutHours,
			}
			err := appSettings.Validate()
			if tc.ExpectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}

}
