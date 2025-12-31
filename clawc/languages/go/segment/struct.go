package segment

import (
	"fmt"
	"io"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/gostdlib/base/context"
)

// fieldEntry tracks where a field is located in the segment.
type fieldEntry struct {
	offset uint32 // Offset in segment data where field starts (0 = at struct header, so offset 8+ for fields)
	size   uint32 // Total size of field (header + data)
	isSet  bool   // Whether field was explicitly set (for IsSet tracking)
}

// Struct represents a Claw struct with data stored directly in wire format.
// Fields are written directly to the segment in sorted field number order.
type Struct struct {
	// seg is the segment containing wire format data.
	seg *Segment
	// mapping is the field mapping for this struct.
	mapping *mapping.Map
	// fieldIndex tracks where each field is located in the segment.
	fieldIndex []fieldEntry
	// fieldIndexParsed indicates whether parseFieldIndex has been called.
	// Used for lazy parsing optimization.
	fieldIndexParsed bool
	// parent tracks the parent.
	parent *Struct
	// parentFN is the field number in the parent struct where this struct is embedded.
	parentFN uint16

	// dirtyLists tracks lists that need to be synced before Marshal.
	dirtyLists []dirtyList

	// lists tracks lazily-created lists for fields.
	lists map[uint16]any

	// isSetEnabled indicates whether IsSet tracking is enabled.
	isSetEnabled bool
	// isSetBits holds the IsSet bitfield bytes.
	isSetBits []byte

	// recording indicates whether mutation recording is enabled for patch generation.
	recording bool
	// recordedOps stores the recorded operations when recording is enabled.
	recordedOps []RecordedOp
}

// dirtyList tracks a list that needs to be synced to the segment before Marshal.
type dirtyList struct {
	fieldNum uint16
	list     ListSyncer
}

// ListSyncer is implemented by list types to sync their data to the parent struct.
type ListSyncer interface {
	SyncToSegment() error
}

// RecordedOp stores a single recorded operation for patch generation.
// This is stored as raw data to avoid import cycles with the patch/msgs package.
type RecordedOp struct {
	FieldNum uint16
	OpType   uint8 // Matches msgs.OpType values
	Index    int32 // List index, -1 for non-list ops
	Data     []byte
}

// OpType constants matching msgs.OpType values.
// Defined here to avoid import cycle with patch/msgs package.
const (
	OpUnknown         uint8 = 0
	OpSet             uint8 = 1
	OpClear           uint8 = 2
	OpStructPatch     uint8 = 3
	OpListReplace     uint8 = 4
	OpListSet         uint8 = 5
	OpListInsert      uint8 = 6
	OpListRemove      uint8 = 7
	OpListStructPatch uint8 = 8
	OpMapSet          uint8 = 9
	OpMapDelete       uint8 = 10
)

// NoListIndex is the sentinel value for non-list operations.
const NoListIndex int32 = -1

// New creates a new Struct with the given mapping.
// The struct is initialized with an 8-byte header.
func New(ctx context.Context, m *mapping.Map) *Struct {
	s := &Struct{
		seg:        SegmentPool.Get(ctx, m),
		mapping:    m,
		fieldIndex: FieldIndexPool.Get(ctx, m),
	}

	// Initialize struct header (field 0, type Struct, size 8)
	EncodeHeader(s.seg.data[0:8], 0, field.FTStruct, 8)

	return s
}

// Mapping returns the field mapping for this struct.
func (s *Struct) Mapping() *mapping.Map {
	return s.mapping
}

// Size returns the current total size of the struct in bytes.
func (s *Struct) Size() int {
	return s.seg.Len()
}

// SetIsSetEnabled enables or disables IsSet tracking.
func (s *Struct) SetIsSetEnabled(enabled bool) {
	s.isSetEnabled = enabled
	if enabled && s.isSetBits == nil {
		// Allocate isSet bits: 1 byte per 7 fields, padded to 8 bytes
		numBytes := (len(s.mapping.Fields) + 6) / 7
		if numBytes == 0 {
			numBytes = 1
		}
		// Pad to 8-byte alignment
		padded := ((numBytes + 7) / 8) * 8
		s.isSetBits = make([]byte, padded)
	}
}

// markFieldSet marks a field as explicitly set (for IsSet tracking).
func (s *Struct) markFieldSet(fieldNum uint16) {
	if s.isSetEnabled && s.isSetBits != nil {
		byteIdx := fieldNum / 7
		bitIdx := fieldNum % 7
		if int(byteIdx) < len(s.isSetBits) {
			s.isSetBits[byteIdx] |= 1 << bitIdx
		}
	}
}

// findInsertPosition finds the offset where a field should be inserted
// to maintain sorted field number order.
func (s *Struct) findInsertPosition(fieldNum uint16) int {
	// Start after the struct header
	pos := HeaderSize

	// Find position after all fields with lower fieldNum
	for i := uint16(0); i < fieldNum; i++ {
		entry := s.fieldIndex[i]
		if entry.size > 0 {
			pos = int(entry.offset) + int(entry.size)
		}
	}

	return pos
}

// updateFieldOffsets updates all field offsets after a shift operation.
// delta is positive for insertions, negative for removals.
func (s *Struct) updateFieldOffsets(afterOffset int, delta int) {
	for i := range s.fieldIndex {
		entry := &s.fieldIndex[i]
		if entry.size > 0 && int(entry.offset) >= afterOffset {
			entry.offset = uint32(int(entry.offset) + delta)
		}
	}
}

// insertField inserts or replaces a field in the segment.
// The field data includes the 8-byte header followed by any additional data.
func (s *Struct) insertField(fieldNum uint16, data []byte) {
	s.ensureFieldIndexParsed()
	if int(fieldNum) >= len(s.fieldIndex) {
		panic(fmt.Sprintf("segment: field number %d out of range", fieldNum))
	}

	totalSize := len(data)
	existing := s.fieldIndex[fieldNum]

	if existing.size > 0 {
		// Replace existing field
		sizeDelta := totalSize - int(existing.size)

		if sizeDelta == 0 {
			// Same size: overwrite in place
			copy(s.seg.data[existing.offset:], data)
		} else {
			// Different size: use ReplaceAt
			s.seg.ReplaceAt(int(existing.offset), int(existing.size), data)
			// Update offsets of fields after this one
			s.updateFieldOffsets(int(existing.offset)+int(existing.size), sizeDelta)
		}

		s.fieldIndex[fieldNum] = fieldEntry{
			offset: existing.offset,
			size:   uint32(totalSize),
			isSet:  true,
		}
	} else {
		// Insert new field at correct sorted position
		insertPos := s.findInsertPosition(fieldNum)

		s.seg.InsertAt(insertPos, data)

		// Update offsets of fields after insert position
		s.updateFieldOffsets(insertPos, totalSize)

		s.fieldIndex[fieldNum] = fieldEntry{
			offset: uint32(insertPos),
			size:   uint32(totalSize),
			isSet:  true,
		}
	}

	// Update struct header size
	s.updateHeaderSize()

	// Propagate size change to parent if any
	s.propagateSizeToParent()
}

// removeField removes a field from the segment.
func (s *Struct) removeField(fieldNum uint16) {
	s.ensureFieldIndexParsed()
	if int(fieldNum) >= len(s.fieldIndex) {
		return
	}

	existing := s.fieldIndex[fieldNum]
	if existing.size == 0 {
		return // Not present
	}

	// Remove the field data
	s.seg.RemoveAt(int(existing.offset), int(existing.size))

	// Update offsets of fields after removed one
	s.updateFieldOffsets(int(existing.offset)+int(existing.size), -int(existing.size))

	// Clear the field entry
	s.fieldIndex[fieldNum] = fieldEntry{}

	// Update struct header size
	s.updateHeaderSize()

	// Propagate size change to parent
	s.propagateSizeToParent()
}

// updateHeaderSize updates the struct header with the current total size.
// Note: This only includes the current segment size, NOT the isSetBits.
// The isSetBits are added during Marshal() and the header is updated then.
func (s *Struct) updateHeaderSize() {
	size := uint64(s.seg.Len())
	EncodeHeaderFinal40(s.seg.data[0:8], size)
}

// propagateSizeToParent updates parent struct sizes when this struct changes.
func (s *Struct) propagateSizeToParent() {
	if s.parent == nil {
		return
	}

	// Re-embed this struct in parent with new size
	// This is called after our size changed, so parent needs to update
	// For now, we'll handle this by requiring finalize before Marshal
}

// RegisterDirtyList registers a list that needs to be synced before Marshal.
func (s *Struct) RegisterDirtyList(fieldNum uint16, list ListSyncer) {
	s.dirtyLists = append(s.dirtyLists, dirtyList{fieldNum: fieldNum, list: list})
	// Also add to lists map so GetList can find it
	s.SetList(fieldNum, list)
}

// GetList returns a lazily-created list for a field, or nil if not yet created.
func (s *Struct) GetList(fieldNum uint16) any {
	if s.lists == nil {
		return nil
	}
	return s.lists[fieldNum]
}

// SetList stores a list object for a field.
func (s *Struct) SetList(fieldNum uint16, list any) {
	if s.lists == nil {
		s.lists = make(map[uint16]any)
	}
	s.lists[fieldNum] = list
}

// ClearListCache removes a cached list for a field.
// This is used when creating uncached lists to ensure subsequent
// accesses read from segment data instead of returning stale cache.
func (s *Struct) ClearListCache(fieldNum uint16) {
	if s.lists != nil {
		delete(s.lists, fieldNum)
	}
}

// syncDirtyLists syncs all dirty lists to the segment.
func (s *Struct) syncDirtyLists() error {
	for _, dl := range s.dirtyLists {
		if err := dl.list.SyncToSegment(); err != nil {
			return err
		}
	}
	s.dirtyLists = s.dirtyLists[:0] // Clear the list
	return nil
}

// SyncDirtyListsForField syncs any dirty lists for a specific field number
// and removes them from the dirty lists.
// This is used before reading field data to ensure segment is up to date.
func (s *Struct) SyncDirtyListsForField(fieldNum uint16) {
	// Sync and remove entries for this field
	newDirtyLists := s.dirtyLists[:0]
	for _, dl := range s.dirtyLists {
		if dl.fieldNum == fieldNum {
			dl.list.SyncToSegment()
			// Don't add back to newDirtyLists (removed)
		} else {
			newDirtyLists = append(newDirtyLists, dl)
		}
	}
	s.dirtyLists = newDirtyLists
}

// removeDirtyListsForField removes any dirty list entries for a specific field
// without syncing. This is used when clearing a field.
func (s *Struct) removeDirtyListsForField(fieldNum uint16) {
	newDirtyLists := s.dirtyLists[:0]
	for _, dl := range s.dirtyLists {
		if dl.fieldNum != fieldNum {
			newDirtyLists = append(newDirtyLists, dl)
		}
	}
	s.dirtyLists = newDirtyLists
}

// appendIsSetBitfield appends the IsSet bitfield to the segment.
func (s *Struct) appendIsSetBitfield() {
	if !s.isSetEnabled || s.isSetBits == nil {
		return
	}

	// Set continuation bits for all but the last actual data byte
	numBytes := (len(s.mapping.Fields) + 6) / 7
	if numBytes == 0 {
		numBytes = 1
	}
	for i := 0; i < numBytes-1; i++ {
		s.isSetBits[i] |= 0x80
	}

	s.seg.Append(s.isSetBits)
}

// MarshalWriter writes the struct to an io.Writer.
// This syncs any dirty lists, appends IsSet bitfield if enabled,
// and writes the segment bytes directly.
func (s *Struct) MarshalWriter(w io.Writer) (int, error) {
	// Sync all dirty lists
	if err := s.syncDirtyLists(); err != nil {
		return 0, err
	}

	// Append IsSet bitfield if enabled (only once, check if already appended)
	// For now, assume Marshal is called once
	if s.isSetEnabled && s.isSetBits != nil {
		s.appendIsSetBitfield()
		s.isSetEnabled = false // Prevent double-append
	}

	// Update header size one final time
	EncodeHeaderFinal40(s.seg.data[0:8], uint64(s.seg.Len()))

	// Write the segment directly - IT'S ALREADY ENCODED!
	return w.Write(s.seg.data)
}

// Marshal returns the struct as a byte slice. This does not copy the data, it
// returns the internal segment bytes. If you need to modify the *Struct after
// this call, use MarshalSafe() which returns a copy.
func (s *Struct) Marshal() ([]byte, error) {
	// Sync all dirty lists
	if err := s.syncDirtyLists(); err != nil {
		return nil, err
	}

	// Append IsSet bitfield if enabled
	if s.isSetEnabled && s.isSetBits != nil {
		s.appendIsSetBitfield()
		s.isSetEnabled = false
	}

	// Update header size
	EncodeHeaderFinal40(s.seg.data[0:8], uint64(s.seg.Len()))

	// Return a copy of the segment data
	result := make([]byte, s.seg.Len())
	copy(result, s.seg.data)
	return result, nil
}

// MarshalSafe returns a copy of the struct's byte slice.
func (s *Struct) MarshalSafe() ([]byte, error) {
	b, err := s.Marshal()
	if err != nil {
		return nil, err
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c, nil
}

// SegmentBytes returns the raw segment bytes without copying.
// The caller must not modify the returned slice.
func (s *Struct) SegmentBytes() []byte {
	return s.seg.data
}

// Unmarshal unmarshals without copying the data buffer. data should not be used again after this call.
// The struct must have a mapping set (either via New() or Init()).
func (s *Struct) Unmarshal(data []byte) error {
	if len(data) < HeaderSize {
		return fmt.Errorf("segment: data too short for header")
	}

	_, fieldType, totalSize := DecodeHeader(data[0:HeaderSize])
	if fieldType != field.FTStruct {
		return fmt.Errorf("segment: expected struct type, got %v", fieldType)
	}

	if int(totalSize) > len(data) {
		return fmt.Errorf("segment: header size %d exceeds data length %d", totalSize, len(data))
	}

	// NO COPY - directly reference the input slice
	s.seg.data = data[:totalSize]

	// Field index is parsed lazily on first field access
	s.fieldIndexParsed = false

	return nil
}

// UnmarshalReader unmarshals a struct from an io.Reader.
// The struct must have a mapping set (either via New() or Init()).
func (s *Struct) UnmarshalReader(r io.Reader) (int, error) {
	// Read the header first to get total size
	header := make([]byte, HeaderSize)
	n, err := io.ReadFull(r, header)
	if err != nil {
		return n, fmt.Errorf("segment: failed to read header: %w", err)
	}

	// Decode header to get total size
	_, fieldType, totalSize := DecodeHeader(header)
	if fieldType != field.FTStruct {
		return n, fmt.Errorf("segment: expected struct type, got %v", fieldType)
	}

	// Reuse existing capacity if possible, otherwise allocate
	if cap(s.seg.data) >= int(totalSize) {
		s.seg.data = s.seg.data[:totalSize]
	} else {
		s.seg.data = make([]byte, totalSize)
	}
	copy(s.seg.data, header)

	// Read the rest of the data
	if totalSize > HeaderSize {
		remaining := int(totalSize) - HeaderSize
		m, err := io.ReadFull(r, s.seg.data[HeaderSize:])
		n += m
		if err != nil {
			return n, fmt.Errorf("segment: failed to read struct body: %w", err)
		}
		if m != remaining {
			return n, fmt.Errorf("segment: short read: got %d, want %d", m, remaining)
		}
	}

	// Field index is parsed lazily on first field access
	s.fieldIndexParsed = false

	return n, nil
}

// ensureFieldIndexParsed parses the field index lazily on first access.
func (s *Struct) ensureFieldIndexParsed() {
	if !s.fieldIndexParsed {
		parseFieldIndex(s)
		s.fieldIndexParsed = true
	}
}

// HasField returns true if the field is present in the segment.
func (s *Struct) HasField(fieldNum uint16) bool {
	s.ensureFieldIndexParsed()
	if int(fieldNum) >= len(s.fieldIndex) {
		return false
	}
	return s.fieldIndex[fieldNum].size > 0
}

// FieldOffset returns the offset and size of a field, or (0, 0) if not present.
func (s *Struct) FieldOffset(fieldNum uint16) (offset, size int) {
	s.ensureFieldIndexParsed()
	if int(fieldNum) >= len(s.fieldIndex) {
		return 0, 0
	}
	entry := s.fieldIndex[fieldNum]
	return int(entry.offset), int(entry.size)
}

// SegmentData returns the raw segment data bytes.
func (s *Struct) SegmentData() []byte {
	return s.seg.data
}

// Release returns the struct to the segment pool.
func (s *Struct) Release(ctx context.Context) {
	if s.seg != nil {
		SegmentPool.Put(ctx, s.mapping, s.seg)
	}
	if s.fieldIndex != nil {
		FieldIndexPool.Put(ctx, s.mapping, s.fieldIndex)
	}
}

// Reset implements the Resetter interface for sync.Pool.
// Called automatically by the pool on Put(). Clears all state for reuse.
func (s *Struct) Reset() {
	s.seg = nil
	s.fieldIndex = nil
	s.fieldIndexParsed = false

	// Clear references for GC
	s.mapping = nil
	s.parent = nil
	s.parentFN = 0
	s.dirtyLists = s.dirtyLists[:0]
	s.lists = nil
	s.isSetEnabled = false
	s.isSetBits = nil
	s.recording = false
	s.recordedOps = s.recordedOps[:0]
}

// Init initializes the struct with a mapping after getting from pool.
func (s *Struct) Init(m *mapping.Map) {
	s.mapping = m

	// If no segment, create one (for non-pooled usage)
	if s.seg == nil {
		s.seg = NewSegment(64)
	}

	// If no field index, create one (for non-pooled usage)
	// Otherwise, fieldIndex was already set by NewPooled() from the pool
	if s.fieldIndex == nil {
		s.fieldIndex = make([]fieldEntry, len(m.Fields))
	}

	// Initialize struct header (field 0, type Struct, size 8)
	EncodeHeader(s.seg.data[0:8], 0, field.FTStruct, 8)
}

// SetRecording enables or disables mutation recording for patch generation.
// When enabled, all Set* operations and list mutations are recorded.
func (s *Struct) SetRecording(enabled bool) {
	s.recording = enabled
	if enabled && s.recordedOps == nil {
		s.recordedOps = make([]RecordedOp, 0, 8)
	}
}

// Recording returns whether mutation recording is enabled.
func (s *Struct) Recording() bool {
	return s.recording
}

// RecordOp records a single operation. Called by Set* functions when recording is enabled.
func (s *Struct) RecordOp(op RecordedOp) {
	if s.recording {
		s.recordedOps = append(s.recordedOps, op)
	}
}

// DrainRecordedOps returns all recorded operations and clears the internal list.
// The returned slice is safe to modify.
func (s *Struct) DrainRecordedOps() []RecordedOp {
	if len(s.recordedOps) == 0 {
		return nil
	}
	ops := make([]RecordedOp, len(s.recordedOps))
	copy(ops, s.recordedOps)
	s.recordedOps = s.recordedOps[:0]
	return ops
}

// RecordedOpsLen returns the number of recorded operations.
func (s *Struct) RecordedOpsLen() int {
	return len(s.recordedOps)
}
