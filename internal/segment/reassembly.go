package segment

import (
	"bytes"
	"fmt"
	"sync"
)

// Reassembly buffers out-of-order chunks and advances strictly by sequence
// number. The reassembled file is produced only when every expected chunk
// is present and ordered.
type Reassembly struct {
	mu       sync.Mutex
	nextSeq  int
	buffer   map[int][]byte
	received map[int]bool
	final    []byte
	done     bool
}

// NewReassembly creates an empty reassembly starting at sequence 1.
func NewReassembly() *Reassembly {
	return &Reassembly{
		nextSeq:  1,
		buffer:   make(map[int][]byte),
		received: make(map[int]bool),
	}
}

// Add accepts one chunk. Out-of-order chunks wait in the buffer; only the
// contiguous prefix starting at nextSeq is drained into the output, so the
// reassembled bytes always reflect chunk sequence order, never arrival
// order. When the expected sequence is still missing the output does not
// advance, leaving the gap to be filled by a later or retransmitted chunk.
func (r *Reassembly) Add(seq int, data []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return fmt.Errorf("reassembly already finalized")
	}
	if seq < 1 {
		return fmt.Errorf("invalid chunk sequence %d", seq)
	}
	if r.received[seq] {
		return fmt.Errorf("duplicate chunk sequence %d", seq)
	}
	r.received[seq] = true
	r.buffer[seq] = data
	// Drain the contiguous prefix in sequence order. The loop stops as soon
	// as the next expected sequence is absent, so a gap stalls the output
	// until the missing chunk arrives.
	for {
		data, ok := r.buffer[r.nextSeq]
		if !ok {
			break
		}
		r.final = append(r.final, data...)
		delete(r.buffer, r.nextSeq)
		r.nextSeq++
	}
	return nil
}

// Finalize returns the reassembled bytes when every chunk 1..count arrived.
func (r *Reassembly) Finalize(count int) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.done {
		return nil, fmt.Errorf("reassembly already finalized")
	}
	for seq := 1; seq <= count; seq++ {
		if !r.received[seq] {
			return nil, fmt.Errorf("missing chunk sequence %d", seq)
		}
	}
	if len(r.buffer) != 0 {
		return nil, fmt.Errorf("reassembly buffer not empty")
	}
	r.done = true
	return bytes.Clone(r.final), nil
}
