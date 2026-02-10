// Package scheduler provides the request scheduling logic
package scheduler

import (
	"container/heap"
	"time"
)

// scheduledRequest represents a scheduled endpoint request
type scheduledRequest struct {
	endpointIdx int
	nextTime    time.Time
	index       int // heap index
}

// requestHeap implements heap.Interface for scheduling
type requestHeap []*scheduledRequest

func (h requestHeap) Len() int           { return len(h) }
func (h requestHeap) Less(i, j int) bool { return h[i].nextTime.Before(h[j].nextTime) }
func (h requestHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *requestHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*scheduledRequest)
	item.index = n
	*h = append(*h, item)
}

func (h *requestHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*h = old[0 : n-1]
	return item
}

// peek returns the next scheduled request without removing it
func (h *requestHeap) peek() *scheduledRequest {
	if len(*h) == 0 {
		return nil
	}
	return (*h)[0]
}

// newRequestHeap creates a new request heap
func newRequestHeap() *requestHeap {
	h := &requestHeap{}
	heap.Init(h)
	return h
}
