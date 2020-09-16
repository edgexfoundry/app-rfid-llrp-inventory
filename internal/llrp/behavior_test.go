//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package llrp

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestMarshalBehaviorText(t *testing.T) {
	// These tests are really just a sanity check
	// to validate assumptions about json marshaling.
	// They just marshal the interface v to JSON
	// and verify the data matches,
	// then unmarshal that back to a new pointer
	// with the same type as v,
	// and validates it matches the original value.

	tests := []struct {
		name       string
		val        interface{}
		data       []byte
		shouldFail bool
	}{
		{"fast", ScanFast, []byte(`"Fast"`), false},
		{"normal", ScanNormal, []byte(`"Normal"`), false},
		{"deep", ScanDeep, []byte(`"Deep"`), false},
		{"unknownScan", ScanType(501), nil, true},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(testCase.val)
			if testCase.shouldFail {
				if err == nil {
					t.Errorf("expected a marshaling error, but got %v", got)
				}
				return
			}

			if !bytes.Equal(got, testCase.data) {
				t.Errorf("got = %s, want %s", got, testCase.data)
			}

			newInst := reflect.New(reflect.TypeOf(testCase.val))
			ptr := newInst.Interface()
			if err := json.Unmarshal(testCase.data, ptr); err != nil {
				t.Errorf("unmarshaling failed: data = %s, error = %v", testCase.data, err)
				return
			}

			newVal := newInst.Elem().Interface()
			if !reflect.DeepEqual(newVal, testCase.val) {
				t.Errorf("roundtrip failed: got = %+v, want %+v", newVal, testCase.val)
			}
		})
	}
}
