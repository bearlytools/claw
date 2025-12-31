// Package segment provides a segment-based runtime for Claw that writes directly
// to wire format during construction, similar to Cap'n Proto.
package segment

import (
	"encoding/binary"
	"maps"
	"sync/atomic"

	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
)

// SegmentPools manages pools of segments for different mappings.
type SegmentPools struct {
	pools atomic.Pointer[map[*mapping.Map]*sync.Pool[*Segment]]
}

// NewSegmentPools creates a new SegmentPools.
func NewSegmentPools() *SegmentPools {
	m := map[*mapping.Map]*sync.Pool[*Segment]{}
	sp := &SegmentPools{}
	sp.pools.Store(&m)
	return sp
}

// RegisterPool registers a segment pool for the given mapping.
// If the mapping is already registered, this is a no-op.
func (s *SegmentPools) RegisterPool(m *mapping.Map) {
	old := s.pools.Load()
	if _, ok := (*old)[m]; ok {
		return // Already registered
	}
	newMap := maps.Clone(*old)
	newMap[m] = sync.NewPool[*Segment](
		context.Background(),
		m.Name+"_segment_pool",
		func() *Segment {
			return NewSegment(64)
		},
	)
	if !s.pools.CompareAndSwap(old, &newMap) {
		s.RegisterPool(m)
	}
}

// Get retrieves a segment from the pool for the given mapping.
// If no pool is registered for the mapping, one is created automatically.
func (s *SegmentPools) Get(ctx context.Context, m *mapping.Map) *Segment {
	pool, ok := (*s.pools.Load())[m]
	if !ok {
		// Auto-register pool if not exists (useful for tests and dynamic mappings)
		s.RegisterPool(m)
		pool = (*s.pools.Load())[m]
	}
	seg := pool.Get(ctx)
	return seg
}

// Put returns a segment to the pool for the given mapping.
func (s *SegmentPools) Put(ctx context.Context, m *mapping.Map, seg *Segment) {
	pool, ok := (*s.pools.Load())[m]
	if !ok {
		panic("segment: no pool registered for mapping " + m.Name)
	}
	// The pool will call Reset on the segment before putting it back.
	pool.Put(ctx, seg)
}

// SegmentPool is the global segment pool manager.
var SegmentPool = NewSegmentPools()

// FieldIndexPools manages pools of field index slices for different mappings.
// Each mapping gets a pool with slices sized for that mapping's field count.
type FieldIndexPools struct {
	pools atomic.Pointer[map[*mapping.Map]*sync.Pool[[]fieldEntry]]
}

// NewFieldIndexPools creates a new FieldIndexPools.
func NewFieldIndexPools() *FieldIndexPools {
	m := map[*mapping.Map]*sync.Pool[[]fieldEntry]{}
	fp := &FieldIndexPools{}
	fp.pools.Store(&m)
	return fp
}

// RegisterPool registers a field index pool for the given mapping.
// If the mapping is already registered, this is a no-op.
func (f *FieldIndexPools) RegisterPool(m *mapping.Map) {
	old := f.pools.Load()
	if _, ok := (*old)[m]; ok {
		return // Already registered
	}
	numFields := len(m.Fields)
	newMap := maps.Clone(*old)
	newMap[m] = sync.NewPool[[]fieldEntry](
		context.Background(),
		m.Name+"_fieldindex_pool",
		func() []fieldEntry {
			return make([]fieldEntry, numFields)
		},
	)
	if !f.pools.CompareAndSwap(old, &newMap) {
		f.RegisterPool(m)
	}
}

// Get retrieves a field index slice from the pool for the given mapping.
func (f *FieldIndexPools) Get(ctx context.Context, m *mapping.Map) []fieldEntry {
	pool, ok := (*f.pools.Load())[m]
	if !ok {
		f.RegisterPool(m)
		pool = (*f.pools.Load())[m]
	}
	return pool.Get(ctx)
}

// Put returns a field index slice to the pool for the given mapping.
func (f *FieldIndexPools) Put(ctx context.Context, m *mapping.Map, fi []fieldEntry) {
	clear(fi)
	pool, ok := (*f.pools.Load())[m]
	if !ok {
		return
	}
	pool.Put(ctx, fi)
}

// FieldIndexPool is the global field index pool manager.
var FieldIndexPool = NewFieldIndexPools()

func init() {
	// Set the registration function in mapping package to break import cycle.
	mapping.RegisterSegmentPool = func(m *mapping.Map) {
		SegmentPool.RegisterPool(m)
		FieldIndexPool.RegisterPool(m)
	}
}

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
