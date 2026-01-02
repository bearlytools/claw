package segment

import (
	"fmt"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/gostdlib/base/concurrency/sync"
	"github.com/gostdlib/base/context"
)

// AnyHashSize is the size of the SHAKE128 type hash (16 bytes / 128 bits).
const AnyHashSize = 16

// AnyMinSize is the minimum size of an Any field (header + hash).
const AnyMinSize = HeaderSize + AnyHashSize

// Any header layout within Final40 (40 bits):
// - Bits 0-7:   Real type (field.Type, e.g., FTStruct=14)
// - Bits 8-39:  Data size in bytes (32 bits, up to 4GB)
const (
	anyRealTypeShift = 0
	anyRealTypeMask  = 0xFF
	anySizeShift     = 8
	anySizeMask      = 0xFFFFFFFF
)

// anyBuffer is a pooled buffer for encoding Any values.
type anyBuffer struct {
	data []byte
}

// Reset resets the buffer for reuse.
func (b *anyBuffer) Reset() {
	b.data = b.data[:0]
}

// anyBufferPool provides pooled buffers for Any encoding to reduce allocations.
var anyBufferPool = sync.NewPool[*anyBuffer](
	context.Background(),
	"any_buffer_pool",
	func() *anyBuffer {
		return &anyBuffer{
			data: make([]byte, 0, 1024),
		}
	},
)

// getAnyBuffer gets a buffer from the pool.
func getAnyBuffer() *anyBuffer {
	return anyBufferPool.Get(context.Background())
}

// putAnyBuffer returns a buffer to the pool.
func putAnyBuffer(b *anyBuffer) {
	b.Reset()
	anyBufferPool.Put(context.Background(), b)
}

// EncodeAnyHeader writes an Any header to the given buffer.
// The buffer must be at least 8 bytes.
func EncodeAnyHeader(buf []byte, fieldNum uint16, realType field.Type, dataSize uint32) {
	if len(buf) < HeaderSize {
		panic("segment: any header buffer too small")
	}

	// Pack realType and dataSize into final40
	final40 := uint64(realType) << anyRealTypeShift
	final40 |= uint64(dataSize) << anySizeShift

	EncodeHeader(buf, fieldNum, field.FTAny, final40)
}

// DecodeAnyHeader reads Any-specific fields from a header buffer.
// Returns the real type contained in the Any and the data size.
func DecodeAnyHeader(buf []byte) (realType field.Type, dataSize uint32) {
	if len(buf) < HeaderSize {
		panic("segment: any header buffer too small")
	}

	_, _, final40 := DecodeHeader(buf)
	realType = field.Type((final40 >> anyRealTypeShift) & anyRealTypeMask)
	dataSize = uint32((final40 >> anySizeShift) & anySizeMask)
	return
}

// TypeHasher is the interface that all Claw structs implement for Any type support.
// It provides the type's unique SHAKE128 hash for type identification.
type TypeHasher interface {
	XXXTypeHash() [16]byte
}

// StructGetter is the interface for getting the underlying segment.Struct.
type StructGetter interface {
	XXXGetStruct() *Struct
}

// SetAny sets an Any field with a Claw struct value.
// The value must implement both TypeHasher and StructGetter interfaces.
// Returns an error if the value doesn't implement the required interfaces.
func SetAny(s *Struct, fieldNum uint16, value any) error {
	if value == nil {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return nil
	}

	// Get the type hash
	hasher, ok := value.(TypeHasher)
	if !ok {
		return fmt.Errorf("segment: SetAny value must implement TypeHasher interface")
	}
	typeHash := hasher.XXXTypeHash()

	// Get the underlying struct
	getter, ok := value.(StructGetter)
	if !ok {
		return fmt.Errorf("segment: SetAny value must implement StructGetter interface")
	}

	innerStruct := getter.XXXGetStruct()
	if innerStruct == nil {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return nil
	}

	// Get serialized data from inner struct
	innerData := innerStruct.SegmentBytes()

	// Calculate total size and get a pooled buffer
	dataSize := uint32(AnyHashSize + len(innerData))
	totalSize := HeaderSize + int(dataSize)

	buf := getAnyBuffer()
	defer putAnyBuffer(buf)

	// Ensure buffer has enough capacity
	if cap(buf.data) < totalSize {
		buf.data = make([]byte, totalSize)
	} else {
		buf.data = buf.data[:totalSize]
	}

	// Encode header with FTAny and real type (FTStruct) in Final40
	EncodeAnyHeader(buf.data[:HeaderSize], fieldNum, field.FTStruct, dataSize)

	// Copy type hash
	copy(buf.data[HeaderSize:HeaderSize+AnyHashSize], typeHash[:])

	// Copy serialized struct data
	copy(buf.data[HeaderSize+AnyHashSize:], innerData)

	// Make a copy for insertion (buffer will be reused)
	data := make([]byte, totalSize)
	copy(data, buf.data)

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: data[HeaderSize:]})
	}

	return nil
}

// GetAnyRaw returns the raw bytes and type hash of an Any field without decoding.
// Returns (nil, empty hash, false) if the field is not set.
// This is useful for forwarding/proxying scenarios where decoding isn't needed.
func GetAnyRaw(s *Struct, fieldNum uint16) (data []byte, typeHash [16]byte, ok bool) {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 || size < AnyMinSize {
		return nil, [16]byte{}, false
	}

	fieldData := s.seg.data[offset : offset+size]

	// Verify it's an Any type
	_, fieldType, _ := DecodeHeader(fieldData[:HeaderSize])
	if fieldType != field.FTAny {
		return nil, [16]byte{}, false
	}

	// Extract type hash (16 bytes after header)
	copy(typeHash[:], fieldData[HeaderSize:HeaderSize+AnyHashSize])

	// Extract serialized data (everything after hash)
	data = fieldData[HeaderSize+AnyHashSize:]

	return data, typeHash, true
}

// GetAny decodes an Any field into the provided target.
// The target must be a pointer to a Claw struct that implements TypeHasher.
// Returns an error if:
// - The field is not set
// - The target doesn't implement required interfaces
// - The type hash doesn't match the stored value
func GetAny(s *Struct, fieldNum uint16, target any) error {
	if target == nil {
		return fmt.Errorf("segment: GetAny target cannot be nil")
	}

	// Get the target's type hash
	hasher, ok := target.(TypeHasher)
	if !ok {
		return fmt.Errorf("segment: GetAny target must implement TypeHasher interface")
	}
	targetHash := hasher.XXXTypeHash()

	// Get the raw data and stored hash
	data, storedHash, ok := GetAnyRaw(s, fieldNum)
	if !ok {
		return fmt.Errorf("segment: Any field %d is not set", fieldNum)
	}

	// Validate type hash matches
	if targetHash != storedHash {
		return fmt.Errorf("segment: type hash mismatch: stored hash doesn't match target type")
	}

	// Get the target's underlying struct for zero-copy decode
	getter, ok := target.(StructGetter)
	if !ok {
		return fmt.Errorf("segment: GetAny target must implement StructGetter interface")
	}

	targetStruct := getter.XXXGetStruct()
	if targetStruct == nil {
		return fmt.Errorf("segment: GetAny target struct is nil")
	}

	// Zero-copy unmarshal directly into target's segment
	return targetStruct.Unmarshal(data)
}

// SetAnyRaw sets an Any field from raw serialized data and type hash.
// This is used by Ingest to reconstitute Any fields from Walk tokens.
func SetAnyRaw(s *Struct, fieldNum uint16, rawData []byte, typeHash []byte) error {
	if len(rawData) == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return nil
	}

	if len(typeHash) != AnyHashSize {
		return fmt.Errorf("segment: SetAnyRaw type hash must be %d bytes", AnyHashSize)
	}

	// Calculate total size and allocate
	dataSize := uint32(AnyHashSize + len(rawData))
	totalSize := HeaderSize + int(dataSize)
	data := make([]byte, totalSize)

	// Encode header with FTAny and real type (FTStruct) in Final40
	EncodeAnyHeader(data[:HeaderSize], fieldNum, field.FTStruct, dataSize)

	// Copy type hash
	copy(data[HeaderSize:HeaderSize+AnyHashSize], typeHash)

	// Copy serialized struct data
	copy(data[HeaderSize+AnyHashSize:], rawData)

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: data[HeaderSize:]})
	}

	return nil
}

// List Any support

// SetListAny sets a []Any field with a slice of Claw struct values.
// Each value must implement both TypeHasher and StructGetter interfaces.
func SetListAny(s *Struct, fieldNum uint16, values []any) error {
	if len(values) == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return nil
	}

	// Calculate total size needed
	totalDataSize := 0
	for i, v := range values {
		if v == nil {
			return fmt.Errorf("segment: SetListAny value at index %d is nil", i)
		}

		getter, ok := v.(StructGetter)
		if !ok {
			return fmt.Errorf("segment: SetListAny value at index %d must implement StructGetter", i)
		}
		innerStruct := getter.XXXGetStruct()
		if innerStruct == nil {
			return fmt.Errorf("segment: SetListAny value at index %d has nil struct", i)
		}

		// Each item: hash (16 bytes) + struct data
		totalDataSize += AnyHashSize + len(innerStruct.SegmentBytes())
	}

	// Allocate buffer: header + all items
	totalSize := HeaderSize + totalDataSize
	data := make([]byte, totalSize)

	// Encode list header with count in Final40
	EncodeHeader(data[:HeaderSize], fieldNum, field.FTListAny, uint64(len(values)))

	// Encode each item
	offset := HeaderSize
	for _, v := range values {
		hasher := v.(TypeHasher)
		typeHash := hasher.XXXTypeHash()

		getter := v.(StructGetter)
		innerStruct := getter.XXXGetStruct()
		innerData := innerStruct.SegmentBytes()

		// Copy type hash
		copy(data[offset:offset+AnyHashSize], typeHash[:])
		offset += AnyHashSize

		// Copy struct data
		copy(data[offset:offset+len(innerData)], innerData)
		offset += len(innerData)
	}

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: data[HeaderSize:]})
	}

	return nil
}

// GetListAnyLen returns the number of items in a []Any field.
// Returns 0 if the field is not set.
func GetListAnyLen(s *Struct, fieldNum uint16) int {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 || size < HeaderSize {
		return 0
	}

	fieldData := s.seg.data[offset : offset+size]
	_, fieldType, count := DecodeHeader(fieldData[:HeaderSize])
	if fieldType != field.FTListAny {
		return 0
	}

	return int(count)
}

// GetListAnyRaw returns the raw bytes and type hash of an item in a []Any field.
// Returns (nil, empty hash, false) if the field is not set or index is out of bounds.
func GetListAnyRaw(s *Struct, fieldNum uint16, index int) (data []byte, typeHash [16]byte, ok bool) {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 || size < HeaderSize {
		return nil, [16]byte{}, false
	}

	fieldData := s.seg.data[offset : offset+size]
	_, fieldType, count := DecodeHeader(fieldData[:HeaderSize])
	if fieldType != field.FTListAny {
		return nil, [16]byte{}, false
	}

	if index < 0 || index >= int(count) {
		return nil, [16]byte{}, false
	}

	// Scan through items to find the requested index
	pos := HeaderSize
	for i := 0; i < int(count); i++ {
		if pos+AnyHashSize > size {
			return nil, [16]byte{}, false
		}

		// Read type hash
		var itemHash [16]byte
		copy(itemHash[:], fieldData[pos:pos+AnyHashSize])
		pos += AnyHashSize

		// Read struct header to get struct size
		if pos+HeaderSize > size {
			return nil, [16]byte{}, false
		}
		_, _, structSize := DecodeHeader(fieldData[pos : pos+HeaderSize])

		if i == index {
			// Found the item
			if pos+int(structSize) > size {
				return nil, [16]byte{}, false
			}
			return fieldData[pos : pos+int(structSize)], itemHash, true
		}

		pos += int(structSize)
	}

	return nil, [16]byte{}, false
}

// GetListAny decodes an item from a []Any field into the provided target.
// The target must be a pointer to a Claw struct that implements TypeHasher.
func GetListAny(s *Struct, fieldNum uint16, index int, target any) error {
	if target == nil {
		return fmt.Errorf("segment: GetListAny target cannot be nil")
	}

	// Get the target's type hash
	hasher, ok := target.(TypeHasher)
	if !ok {
		return fmt.Errorf("segment: GetListAny target must implement TypeHasher interface")
	}
	targetHash := hasher.XXXTypeHash()

	// Get the raw data and stored hash
	data, storedHash, ok := GetListAnyRaw(s, fieldNum, index)
	if !ok {
		return fmt.Errorf("segment: []Any field %d index %d is not set", fieldNum, index)
	}

	// Validate type hash matches
	if targetHash != storedHash {
		return fmt.Errorf("segment: type hash mismatch at index %d: stored hash doesn't match target type", index)
	}

	// Get the target's underlying struct for zero-copy decode
	getter, ok := target.(StructGetter)
	if !ok {
		return fmt.Errorf("segment: GetListAny target must implement StructGetter interface")
	}

	targetStruct := getter.XXXGetStruct()
	if targetStruct == nil {
		return fmt.Errorf("segment: GetListAny target struct is nil")
	}

	// Zero-copy unmarshal directly into target's segment
	return targetStruct.Unmarshal(data)
}

// AnyRawItem represents a raw Any item for SetListAnyRaw.
type AnyRawItem struct {
	Data     []byte
	TypeHash []byte
}

// SetListAnyRaw sets a []Any field from raw serialized data and type hashes.
// This is used by Ingest to reconstitute []Any fields from Walk tokens.
func SetListAnyRaw(s *Struct, fieldNum uint16, items []AnyRawItem) error {
	if len(items) == 0 {
		s.removeField(fieldNum)
		if s.recording {
			s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpClear, Index: NoListIndex})
		}
		return nil
	}

	// Calculate total size needed
	totalDataSize := 0
	for i, item := range items {
		if len(item.TypeHash) != AnyHashSize {
			return fmt.Errorf("segment: SetListAnyRaw item %d type hash must be %d bytes", i, AnyHashSize)
		}
		totalDataSize += AnyHashSize + len(item.Data)
	}

	// Allocate buffer: header + all items
	totalSize := HeaderSize + totalDataSize
	data := make([]byte, totalSize)

	// Encode list header with count in Final40
	EncodeHeader(data[:HeaderSize], fieldNum, field.FTListAny, uint64(len(items)))

	// Encode each item
	offset := HeaderSize
	for _, item := range items {
		// Copy type hash
		copy(data[offset:offset+AnyHashSize], item.TypeHash)
		offset += AnyHashSize

		// Copy struct data
		copy(data[offset:offset+len(item.Data)], item.Data)
		offset += len(item.Data)
	}

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)

	if s.recording {
		s.RecordOp(RecordedOp{FieldNum: fieldNum, OpType: OpSet, Index: NoListIndex, Data: data[HeaderSize:]})
	}

	return nil
}
