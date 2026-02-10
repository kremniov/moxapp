// Package metrics provides in-memory metrics collection
package metrics

import "sort"

// RingBuffer is a fixed-size circular buffer for storing recent values
type RingBuffer struct {
	data     []float64
	capacity int
	index    int
	size     int
}

// NewRingBuffer creates a new ring buffer with the given capacity
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{
		data:     make([]float64, capacity),
		capacity: capacity,
	}
}

// Add adds a value to the ring buffer
func (rb *RingBuffer) Add(value float64) {
	rb.data[rb.index] = value
	rb.index = (rb.index + 1) % rb.capacity
	if rb.size < rb.capacity {
		rb.size++
	}
}

// Percentile calculates the p-th percentile of the stored values
func (rb *RingBuffer) Percentile(p float64) float64 {
	if rb.size == 0 {
		return 0
	}

	// Copy data for sorting
	sortedData := make([]float64, rb.size)
	copy(sortedData, rb.data[:rb.size])
	sort.Float64s(sortedData)

	index := int(float64(rb.size) * p / 100.0)
	if index >= rb.size {
		index = rb.size - 1
	}

	return sortedData[index]
}

// Max returns the maximum value in the buffer
func (rb *RingBuffer) Max() float64 {
	if rb.size == 0 {
		return 0
	}

	max := rb.data[0]
	for i := 1; i < rb.size; i++ {
		if rb.data[i] > max {
			max = rb.data[i]
		}
	}
	return max
}

// Min returns the minimum value in the buffer
func (rb *RingBuffer) Min() float64 {
	if rb.size == 0 {
		return 0
	}

	min := rb.data[0]
	for i := 1; i < rb.size; i++ {
		if rb.data[i] < min {
			min = rb.data[i]
		}
	}
	return min
}

// Average returns the average of all values in the buffer
func (rb *RingBuffer) Average() float64 {
	if rb.size == 0 {
		return 0
	}

	var sum float64
	for i := 0; i < rb.size; i++ {
		sum += rb.data[i]
	}
	return sum / float64(rb.size)
}

// Size returns the number of values currently stored
func (rb *RingBuffer) Size() int {
	return rb.size
}

// Reset clears all values from the buffer
func (rb *RingBuffer) Reset() {
	rb.index = 0
	rb.size = 0
}

// Values returns a copy of all stored values
func (rb *RingBuffer) Values() []float64 {
	result := make([]float64, rb.size)
	copy(result, rb.data[:rb.size])
	return result
}
