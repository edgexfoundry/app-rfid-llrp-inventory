/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package inventory

// CircularBuffer is essentially a moving slice with a max size, where every time a new value is inserted,
// the oldest value is removed from the slice. This is used for calculating moving averages of values over time.
// For performance reasons it is implemented as a fixed size slice with a pointer to where to insert the next value
// such that no new memory allocations need to be made.
type CircularBuffer struct {
	windowSize int
	values     []float64
	counter    int
}

// NewCircularBuffer allocates memory for a new CircularBuffer with the given windowSize
func NewCircularBuffer(windowSize int) *CircularBuffer {
	return &CircularBuffer{
		windowSize: windowSize,
		values:     make([]float64, windowSize),
	}
}

// GetCount returns the number of actual values present in the buffer
// count can be between 0 and windowSize
func (buff *CircularBuffer) GetCount() int {
	if buff.counter >= buff.windowSize {
		return buff.windowSize
	}
	return buff.counter
}

// GetMean returns the average value of all data points in the backing slice.
// Because this is a circular buffer, this value can be considered as a moving average
func (buff *CircularBuffer) GetMean() float64 {
	count := buff.GetCount()
	var total float64
	for i := 0; i < count; i++ {
		total += buff.values[i]
	}
	return total / float64(count)
}

// AddValue appends a new value onto the backing slice,
// overriding the oldest existing value if count has reached windowSize
func (buff *CircularBuffer) AddValue(value float64) {
	buff.values[buff.counter%buff.windowSize] = value
	buff.counter++
}
