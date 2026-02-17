package logs

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewRingBuffer(t *testing.T) {
	rb := NewRingBuffer(100)
	if rb.Capacity() != 100 {
		t.Errorf("Capacity() = %d, want 100", rb.Capacity())
	}
	if rb.Len() != 0 {
		t.Errorf("Len() = %d, want 0", rb.Len())
	}
	if rb.Dropped() != 0 {
		t.Errorf("Dropped() = %d, want 0", rb.Dropped())
	}
	lines := rb.Lines()
	if lines != nil {
		t.Errorf("Lines() = %v, want nil", lines)
	}
}

func TestNewRingBufferDefaultCapacity(t *testing.T) {
	rb := NewRingBuffer(0)
	if rb.Capacity() != DefaultCapacity {
		t.Errorf("Capacity() = %d, want %d for zero input", rb.Capacity(), DefaultCapacity)
	}

	rb2 := NewRingBuffer(-5)
	if rb2.Capacity() != DefaultCapacity {
		t.Errorf("Capacity() = %d, want %d for negative input", rb2.Capacity(), DefaultCapacity)
	}
}

func TestAppendAndLines(t *testing.T) {
	rb := NewRingBuffer(20)

	for i := 0; i < 10; i++ {
		rb.Append(fmt.Sprintf("line-%d", i))
	}

	if rb.Len() != 10 {
		t.Fatalf("Len() = %d, want 10", rb.Len())
	}
	if rb.Dropped() != 0 {
		t.Errorf("Dropped() = %d, want 0", rb.Dropped())
	}

	lines := rb.Lines()
	if len(lines) != 10 {
		t.Fatalf("len(Lines()) = %d, want 10", len(lines))
	}

	for i, line := range lines {
		expected := fmt.Sprintf("line-%d", i)
		if line != expected {
			t.Errorf("Lines()[%d] = %q, want %q", i, line, expected)
		}
	}
}

func TestOverflow(t *testing.T) {
	rb := NewRingBuffer(10)

	// Append 30 lines to a capacity-10 buffer
	for i := 0; i < 30; i++ {
		rb.Append(fmt.Sprintf("line-%d", i))
	}

	if rb.Len() != 10 {
		t.Fatalf("Len() = %d, want 10", rb.Len())
	}
	if rb.Dropped() != 20 {
		t.Errorf("Dropped() = %d, want 20", rb.Dropped())
	}

	lines := rb.Lines()
	if len(lines) != 10 {
		t.Fatalf("len(Lines()) = %d, want 10", len(lines))
	}

	// Should have lines 20-29 (the last 10)
	for i, line := range lines {
		expected := fmt.Sprintf("line-%d", i+20)
		if line != expected {
			t.Errorf("Lines()[%d] = %q, want %q", i, line, expected)
		}
	}
}

func TestAppendBatch(t *testing.T) {
	// Verify batch append produces same result as individual appends
	rb1 := NewRingBuffer(10)
	rb2 := NewRingBuffer(10)

	batch := make([]string, 15)
	for i := 0; i < 15; i++ {
		batch[i] = fmt.Sprintf("line-%d", i)
		rb1.Append(batch[i])
	}
	rb2.AppendBatch(batch)

	lines1 := rb1.Lines()
	lines2 := rb2.Lines()

	if len(lines1) != len(lines2) {
		t.Fatalf("Batch len = %d, individual len = %d", len(lines2), len(lines1))
	}
	for i := range lines1 {
		if lines1[i] != lines2[i] {
			t.Errorf("index %d: batch = %q, individual = %q", i, lines2[i], lines1[i])
		}
	}
	if rb1.Dropped() != rb2.Dropped() {
		t.Errorf("Dropped: batch = %d, individual = %d", rb2.Dropped(), rb1.Dropped())
	}
}

func TestClear(t *testing.T) {
	rb := NewRingBuffer(10)

	for i := 0; i < 15; i++ {
		rb.Append(fmt.Sprintf("line-%d", i))
	}

	if rb.Len() != 10 {
		t.Fatalf("Len() before clear = %d, want 10", rb.Len())
	}
	if rb.Dropped() != 5 {
		t.Fatalf("Dropped() before clear = %d, want 5", rb.Dropped())
	}

	rb.Clear()

	if rb.Len() != 0 {
		t.Errorf("Len() after clear = %d, want 0", rb.Len())
	}
	if rb.Dropped() != 0 {
		t.Errorf("Dropped() after clear = %d, want 0", rb.Dropped())
	}
	if lines := rb.Lines(); lines != nil {
		t.Errorf("Lines() after clear = %v, want nil", lines)
	}
}

func TestConcurrentAppend(t *testing.T) {
	rb := NewRingBuffer(100)

	var wg sync.WaitGroup
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(goroutine int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				rb.Append(fmt.Sprintf("g%d-line-%d", goroutine, i))
			}
		}(g)
	}
	wg.Wait()

	if rb.Len() > rb.Capacity() {
		t.Errorf("Len() = %d exceeds Capacity() = %d", rb.Len(), rb.Capacity())
	}

	// Total appended = 10 * 100 = 1000, capacity = 100
	// So Len should be exactly 100 and Dropped should be 900
	if rb.Len() != 100 {
		t.Errorf("Len() = %d, want 100", rb.Len())
	}
	if rb.Dropped() != 900 {
		t.Errorf("Dropped() = %d, want 900", rb.Dropped())
	}

	// Verify Lines() returns exactly Len() items without panic
	lines := rb.Lines()
	if len(lines) != rb.Len() {
		t.Errorf("len(Lines()) = %d, Len() = %d", len(lines), rb.Len())
	}
}
