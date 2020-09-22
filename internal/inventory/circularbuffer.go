//
// Copyright (C) 2020 Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package inventory

import (
	"sync"
)

// CircularBuffer is essentially a moving slice with a max size, where every time a new value is inserted,
// the oldest value is removed from the slice. This is used for calculating moving averages of values over time.
// For performance reasons it is implemented as a fixed size slice with a pointer to where to insert the next value
// such that no new memory allocations need to be made.
type CircularBuffer struct {
	values []float64
	total  float64
	index  int
	mutex  sync.RWMutex
}

// NewCircularBuffer allocates memory for a new CircularBuffer with the given windowSize
func NewCircularBuffer(windowSize int) *CircularBuffer {
	if windowSize <= 0 {
		panic("illegal window size")
	}

	return &CircularBuffer{
		values: make([]float64, 0, windowSize),
	}
}

// Len returns the number of actual values present in the buffer
func (buff *CircularBuffer) Len() int {
	buff.mutex.RLock()
	defer buff.mutex.RUnlock()

	return len(buff.values)
}

// Mean returns the average value of all data points in the backing slice.
// Because this is a circular buffer, this value can be considered as a moving average
//
// NOTE: If there is no data in the buffer, this function will return: Nan
func (buff *CircularBuffer) Mean() float64 {
	buff.mutex.RLock()
	defer buff.mutex.RUnlock()

	return buff.total / float64(len(buff.values))
}

// AddValue appends a new value onto the backing slice,
// overriding the oldest existing value if count has reached windowSize
func (buff *CircularBuffer) AddValue(value float64) {
	buff.mutex.Lock()
	defer buff.mutex.Unlock()

	if len(buff.values) < cap(buff.values) {
		buff.values = append(buff.values, value)
		buff.total += value
		return
	}

	// subtract old value and add new value
	buff.total = buff.total - buff.values[buff.index] + value
	// record new value where old was
	buff.values[buff.index] = value

	buff.index++
	if buff.index >= cap(buff.values) {
		// wrap if needed
		buff.index = 0
	}
}
