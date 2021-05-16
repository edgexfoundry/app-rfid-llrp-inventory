//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	// epsilon is used to compare floating point numbers to each other
	epsilon = math.Nextafter(1.0, 2.0) - 1.0
)

func assertBufferSize(t *testing.T, buff *circularBuffer, expectedSize int) {
	assert.Equalf(t, buff.Len(), expectedSize, "expected buffer size of %d, but was %d", buff.Len(), expectedSize)
}

func TestCircularBuffer_AddValue(t *testing.T) {
	windowSizes := []int{1, 5, 10, 20, 100, 999}

	for _, window := range windowSizes {
		window := window
		t.Run(fmt.Sprintf("WindowOf%d", window), func(t *testing.T) {
			buff := newCircularBuffer(window)

			assertBufferSize(t, buff, 0)
			// fill up the buffer
			for i := 0; i < window; i++ {
				buff.AddValue(float64(i))
			}
			assertBufferSize(t, buff, window)

			// attempt to overflow
			for i := 0; i < window*5; i++ {
				buff.AddValue(float64(i))
				// make sure does not overflow
				assertBufferSize(t, buff, window)
			}
		})
	}
}

func TestCircularBuffer_GetMean(t *testing.T) {
	tests := []struct {
		name     string
		window   int
		data     []float64
		expected float64
	}{
		{
			name:     "Basic",
			window:   10,
			data:     []float64{1, 2, 3, 4, 5},
			expected: 3,
		},
		{
			name:     "Basic 2",
			window:   100,
			data:     []float64{10, 20},
			expected: 15,
		},
		{
			name:     "Circular Overflow",
			window:   2,
			data:     []float64{5, 20, 20},
			expected: 20,
		},
		{
			name:     "Circular Overflow 2",
			window:   3,
			data:     []float64{5, 5, 5, 5, 5, 5, 5, 5, 6, 100},
			expected: 37,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			buff := newCircularBuffer(test.window)
			for _, val := range test.data {
				buff.AddValue(val)
			}

			mean := buff.Mean()
			assert.LessOrEqualf(t, math.Abs(mean-test.expected), epsilon, "expected mean of %v, but got %v", test.expected, mean)
		})
	}
}

func TestCircularBuffer_GetCount(t *testing.T) {
	tests := []struct {
		name          string
		windowSize    int
		numberToAdd   uint64
		expectedCount int
	}{
		{
			name:          "Below Window Size",
			windowSize:    20,
			numberToAdd:   1,
			expectedCount: 1,
		},
		{
			name:          "Above Window Size",
			windowSize:    20,
			numberToAdd:   100,
			expectedCount: 20,
		},
		{
			name:          "Exactly Window Size",
			windowSize:    20,
			numberToAdd:   20,
			expectedCount: 20,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			buff := newCircularBuffer(test.windowSize)
			var i uint64
			for i = 0; i < test.numberToAdd; i++ {
				buff.AddValue(1.0)
			}

			count := buff.Len()
			assert.Equalf(t, count, test.expectedCount, "buff.Len() returned %d, but we expected %d", count, test.expectedCount)
		})
	}
}

func TestCircularBuffer_Wrap(t *testing.T) {
	windowSize := 10
	buff := newCircularBuffer(windowSize)

	assertBufferSize(t, buff, 0)
	// fill up the buffer
	for i := 0; i < 10*windowSize; i++ {
		val := float64(i * 2)
		buff.AddValue(val)
		assertBufferSize(t, buff, int(math.Min(float64(i+1), float64(windowSize))))
		assert.Equal(t, buff.values[i%windowSize], val, "A value was added in the wrong location!")
	}
	assertBufferSize(t, buff, windowSize)
}
