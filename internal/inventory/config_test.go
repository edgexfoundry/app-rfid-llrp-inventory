//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"github.com/stretchr/testify/require"
	"math"
	"reflect"
	"strconv"
	"testing"
	"testing/quick"
)

func TestEmptyConfigDefaults(t *testing.T) {
	conf, err := ParseConsulConfig(getTestingLogger(), map[string]string{})
	require.NoError(t, err)

	expected := NewConsulConfig()
	require.Equal(t, expected, conf)
}

func TestParseConsulConfig(t *testing.T) {
	type testCase struct {
		key, val string
		err      error
		exp      interface{}
	}

	cases := []testCase{
		{key: "", val: "600", err: ErrUnexpectedConfigItems},
		{key: "foo", val: "bar", err: ErrUnexpectedConfigItems},

		{key: "AdjustLastReadOnByOrigin", val: "TRUE", exp: true},
		{key: "AdjustLastReadOnByOrigin", val: "true", exp: true},
		{key: "AdjustLastReadOnByOrigin", val: "FALSE", exp: false},
		{key: "AdjustLastReadOnByOrigin", val: "false", exp: false},
		{key: "AdjustLastReadOnByOrigin", val: "no", err: strconv.ErrSyntax},
		{key: "AdjustLastReadOnByOrigin", val: "yes", err: strconv.ErrSyntax},
		{key: "AdjustLastReadOnByOrigin", val: "", err: strconv.ErrSyntax},
		{key: "AdjustLastReadOnByOrigin", val: "", err: strconv.ErrSyntax},

		{key: "DepartedThresholdSeconds", val: "600", exp: uint(600)},
		{key: "DepartedThresholdSeconds", val: "1000", exp: uint(1000)},
		{key: "DepartedThresholdSeconds", val: "18446744073709551615", exp: uint(18446744073709551615)},
		{key: "DepartedThresholdSeconds", val: "0", err: ErrOutOfRange},
		{key: "DepartedThresholdSeconds", val: "18446744073709551616", err: strconv.ErrRange},
		{key: "DepartedThresholdSeconds", val: "-600", err: strconv.ErrSyntax},
		{key: "DepartedThresholdSeconds", val: "10.6", err: strconv.ErrSyntax},
		{key: "DepartedThresholdSeconds", val: "", err: strconv.ErrSyntax},
		{key: "DepartedThresholdSeconds", val: "  ", err: strconv.ErrSyntax},
		{key: "DepartedThresholdSeconds", val: "asdf", err: strconv.ErrSyntax},

		{key: "DepartedCheckIntervalSeconds", val: "600", exp: uint(600)},
		{key: "DepartedCheckIntervalSeconds", val: "18446744073709551615", exp: uint(18446744073709551615)},
		{key: "DepartedCheckIntervalSeconds", val: "0", err: ErrOutOfRange},
		{key: "DepartedCheckIntervalSeconds", val: "-600", err: strconv.ErrSyntax},
		{key: "DepartedCheckIntervalSeconds", val: "6.00", err: strconv.ErrSyntax},
		{key: "DepartedCheckIntervalSeconds", val: "99999999999999999999999", err: strconv.ErrRange},

		{key: "AgeOutHours", val: "600", exp: uint(600)},
		{key: "AgeOutHours", val: "18446744073709551615", exp: uint(18446744073709551615)},
		{key: "AgeOutHours", val: "0", err: ErrOutOfRange},
		{key: "AgeOutHours", val: "-600", err: strconv.ErrSyntax},
		{key: "AgeOutHours", val: "6.00", err: strconv.ErrSyntax},
		{key: "AgeOutHours", val: "99999999999999999999999", err: strconv.ErrRange},

		{key: "MobilityProfileThreshold", val: "5.0", exp: float64(5.0)},
		{key: "MobilityProfileThreshold", val: "600", exp: float64(600)},
		{key: "MobilityProfileThreshold", val: "-600", exp: float64(-600)},

		{key: "MobilityProfileHoldoffMillis", val: "1250.0", exp: float64(1250.0)},
		{key: "MobilityProfileHoldoffMillis", val: "600", exp: float64(600)},
		{key: "MobilityProfileHoldoffMillis", val: "-600", exp: float64(-600)},

		{key: "MobilityProfileSlope", val: "-0.0055", exp: float64(-0.0055)},

		{key: "DeviceServiceName", val: "testing", exp: "testing"},
		{key: "DeviceServiceName", val: "", exp: ""},
		{key: "DeviceServiceName", val: " ", exp: " "},

		{key: "DeviceServiceURL", val: "http://testing:49989/", exp: "http://testing:49989/"},
		{key: "DeviceServiceURL", val: "", exp: ""},
		{key: "MetadataServiceURL", val: "", exp: ""},
	}

	rt := reflect.TypeOf(ApplicationSettings{})
	for _, c := range cases {
		c := c
		t.Run(c.key+":"+c.val, func(tt *testing.T) {
			cfgMap := map[string]string{c.key: c.val}
			ccfg, err := ParseConsulConfig(getTestingLogger(), cfgMap)
			require.ErrorIs(t, err, c.err)
			if c.err != nil {
				return
			}

			ft, ok := rt.FieldByName(c.key)
			require.True(t, ok)

			rv := reflect.ValueOf(ccfg.ApplicationSettings)
			fv := rv.FieldByIndex(ft.Index)
			require.True(t, fv.IsValid())
			require.True(t, fv.CanInterface())
			require.Equal(t, fv.Interface(), c.exp)
		})
	}
}

func TestQuickCheckStr(t *testing.T) {
	// quick.Check that we return an error (and don't panic) on arbitrary strings.
	t.Run("quickCheckStr", func(tt *testing.T) {
		tt.Parallel()
		err := quick.Check(func(val string) bool {
			conf, parseErr := ParseConsulConfig(nil, map[string]string{
				"DeviceServiceName": val})
			return parseErr == nil && conf.ApplicationSettings.DeviceServiceName == val
		}, nil);
		require.NoError(tt, err)
	})
}

func TestQuickCheckUint(t *testing.T) {
	// quick.Check that we can round-trip arbitrary uints.
	t.Run("quickCheckUint", func(tt *testing.T) {
		tt.Parallel()
		//if err := quick.Check(func(u uint) bool {
		//	if u == 0 {
		//		return true
		//	}
		//	iStr := strconv.FormatUint(uint64(u), 10)
		//	conf, parseErr := ParseConsulConfig(nil, map[string]string{"AgeOutHours": iStr})
		//	return parseErr == nil && conf.ApplicationSettings.AgeOutHours == u
		//}, nil); err != nil {
		//	t.Error(err)
		//}
		err := quick.Check(func(u uint) bool {
			if u == 0 {
				return true
			}
			iStr := strconv.FormatUint(uint64(u), 10)
			conf, parseErr := ParseConsulConfig(nil, map[string]string{"AgeOutHours": iStr})
			return parseErr == nil && conf.ApplicationSettings.AgeOutHours == u
		}, nil);
		require.NoError(tt, err)
	})
}

func TestQuickCheckFloat64(t *testing.T) {
	// quick.Check that we can round-trip arbitrary float64s.
	t.Run("quickCheckFloat64", func(tt *testing.T) {
		tt.Parallel()
		// Note: FormatFloat can produce binary values (with 'b'),
		// but ParseFloat can't parse them, so it's not in this list.
		fmts := [...]byte{'e', 'E', 'f', 'g', 'G', 'x', 'X'}
		err := quick.Check(func(f float64, fmtByte byte) bool {
			iStr := strconv.FormatFloat(f, fmts[int(fmtByte)%len(fmts)], -1, 64)
			conf, parseErr := ParseConsulConfig(nil, map[string]string{
				"MobilityProfileThreshold": iStr})
			if parseErr != nil {
				tt.Logf("fmt: %c (%d), f: %v, err: %+v",
					fmts[int(fmtByte)%len(fmts)], fmtByte, f, parseErr)
			}
			return parseErr == nil && math.Abs(
				conf.ApplicationSettings.MobilityProfileThreshold-f) < 0.001
		}, nil);
		require.NoError(tt, err)
	})
}