package logs

import "sync"

// DefaultCapacity is the default maximum number of lines the ring buffer holds.
const DefaultCapacity = 5000

// RingBuffer is a fixed-capacity circular buffer for log lines.
// It provides O(1) append and tracks how many lines have been dropped
// due to overflow. All methods are safe for concurrent use.
type RingBuffer struct {
	lines    []string
	head     int   // index of the oldest element
	count    int   // current number of stored elements
	capacity int   // maximum number of elements
	dropped  int64 // total number of lines dropped due to overflow
	mu       sync.Mutex
}

// NewRingBuffer creates a ring buffer with the given maximum capacity.
func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = DefaultCapacity
	}
	return &RingBuffer{
		lines:    make([]string, capacity),
		capacity: capacity,
	}
}

// Append adds a single line to the buffer. When the buffer is full,
// the oldest line is overwritten and the dropped counter is incremented.
// This operation is O(1) and thread-safe.
func (rb *RingBuffer) Append(line string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count < rb.capacity {
		// Buffer not full yet -- write at (head + count) mod capacity
		idx := (rb.head + rb.count) % rb.capacity
		rb.lines[idx] = line
		rb.count++
	} else {
		// Buffer full -- overwrite oldest at head
		rb.lines[rb.head] = line
		rb.head = (rb.head + 1) % rb.capacity
		rb.dropped++
	}
}

// AppendBatch adds multiple lines efficiently with a single lock acquisition.
func (rb *RingBuffer) AppendBatch(lines []string) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	for _, line := range lines {
		if rb.count < rb.capacity {
			idx := (rb.head + rb.count) % rb.capacity
			rb.lines[idx] = line
			rb.count++
		} else {
			rb.lines[rb.head] = line
			rb.head = (rb.head + 1) % rb.capacity
			rb.dropped++
		}
	}
}

// Lines returns all stored lines in chronological order (oldest to newest).
// A new slice is allocated; the caller owns the returned data.
func (rb *RingBuffer) Lines() []string {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil
	}

	result := make([]string, rb.count)
	for i := 0; i < rb.count; i++ {
		result[i] = rb.lines[(rb.head+i)%rb.capacity]
	}
	return result
}

// Len returns the current number of stored lines.
func (rb *RingBuffer) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}

// Dropped returns the total number of lines that have been dropped
// due to buffer overflow since creation or last Clear.
func (rb *RingBuffer) Dropped() int64 {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.dropped
}

// Clear resets the buffer to empty state and zeroes the dropped counter.
func (rb *RingBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.head = 0
	rb.count = 0
	rb.dropped = 0
	// Zero out the slice to allow GC of old strings
	for i := range rb.lines {
		rb.lines[i] = ""
	}
}

// Capacity returns the maximum number of lines the buffer can hold.
func (rb *RingBuffer) Capacity() int {
	return rb.capacity
}
