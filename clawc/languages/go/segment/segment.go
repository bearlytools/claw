// Package segment provides a segment-based runtime for Claw that writes directly
// to wire format during construction, similar to Cap'n Proto.
package segment

import (
	"encoding/binary"
)

// Segment is a contiguous byte buffer that IS the wire format.
// All field writes go directly into this buffer in sorted field order.
type Segment struct {
	data []byte // The wire format bytes
}

// NewSegment creates a new Segment with the given initial capacity in bytes.
// Anything less than 8 bytes will be rounded up to a minimum reasonable size.
// The segment starts with 8 bytes for the struct header for len, regardless of capacity.
func NewSegment(capacity int) *Segment {
	if capacity < 64 {
		capacity = 64 // Minimum reasonable size
	}
	s := &Segment{
		data: make([]byte, 8, capacity),
	}
	return s
}

// Len returns the current length of valid data in the segment.
func (s *Segment) Len() int {
	return len(s.data)
}

// Cap returns the capacity of the segment.
func (s *Segment) Cap() int {
	return cap(s.data)
}

// Bytes returns the segment data. The returned slice should not be modified.
func (s *Segment) Bytes() []byte {
	return s.data
}

// Reset clears the segment data but keeps capacity.
func (s *Segment) Reset() {
	s.data = s.data[:8] // Keep header space
	clear(s.data[:8])
}

// EnsureCapacity ensures the segment can hold at least n more bytes.
func (s *Segment) EnsureCapacity(n int) {
	needed := len(s.data) + n
	if needed <= cap(s.data) {
		return
	}
	newCap := max(cap(s.data)*2, needed)
	newData := make([]byte, len(s.data), newCap)
	copy(newData, s.data)
	s.data = newData
}

// Append appends data to the end of the segment.
func (s *Segment) Append(data []byte) {
	s.EnsureCapacity(len(data))
	s.data = append(s.data, data...)
}

// InsertAt inserts data at the given offset, shifting existing data to make room.
// The offset must be within [0, Len()].
func (s *Segment) InsertAt(offset int, data []byte) {
	if offset < 0 || offset > len(s.data) {
		panic("segment: insert offset out of bounds")
	}
	if len(data) == 0 {
		return
	}

	s.EnsureCapacity(len(data))

	oldLen := len(s.data)
	s.data = s.data[:oldLen+len(data)]

	if offset < oldLen {
		copy(s.data[offset+len(data):], s.data[offset:oldLen])
	}

	copy(s.data[offset:], data)
}

// RemoveAt removes n bytes starting at offset, shifting subsequent data.
func (s *Segment) RemoveAt(offset, n int) {
	if offset < 0 || offset+n > len(s.data) {
		panic("segment: remove range out of bounds")
	}
	if n == 0 {
		return
	}

	copy(s.data[offset:], s.data[offset+n:])
	s.data = s.data[:len(s.data)-n]
}

// ReplaceAt replaces n bytes at offset with newData.
// If len(newData) != n, data is shifted accordingly.
func (s *Segment) ReplaceAt(offset, n int, newData []byte) {
	if offset < 0 || offset+n > len(s.data) {
		panic("segment: replace range out of bounds")
	}

	delta := len(newData) - n
	if delta == 0 {
		copy(s.data[offset:], newData)
		return
	}

	if delta > 0 {
		s.EnsureCapacity(delta)
		oldLen := len(s.data)
		s.data = s.data[:oldLen+delta]
		copy(s.data[offset+len(newData):], s.data[offset+n:oldLen])
	} else {
		copy(s.data[offset+len(newData):], s.data[offset+n:])
		s.data = s.data[:len(s.data)+delta]
	}

	copy(s.data[offset:], newData)
}

// WriteUint16 writes a uint16 at the given offset in little-endian.
func (s *Segment) WriteUint16(offset int, v uint16) {
	binary.LittleEndian.PutUint16(s.data[offset:], v)
}

// WriteUint32 writes a uint32 at the given offset in little-endian.
func (s *Segment) WriteUint32(offset int, v uint32) {
	binary.LittleEndian.PutUint32(s.data[offset:], v)
}

// WriteUint64 writes a uint64 at the given offset in little-endian.
func (s *Segment) WriteUint64(offset int, v uint64) {
	binary.LittleEndian.PutUint64(s.data[offset:], v)
}

// ReadUint16 reads a uint16 from the given offset in little-endian.
func (s *Segment) ReadUint16(offset int) uint16 {
	return binary.LittleEndian.Uint16(s.data[offset:])
}

// ReadUint32 reads a uint32 from the given offset in little-endian.
func (s *Segment) ReadUint32(offset int) uint32 {
	return binary.LittleEndian.Uint32(s.data[offset:])
}

// ReadUint64 reads a uint64 from the given offset in little-endian.
func (s *Segment) ReadUint64(offset int) uint64 {
	return binary.LittleEndian.Uint64(s.data[offset:])
}
