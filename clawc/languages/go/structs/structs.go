// Package structs contains objects, functions and methods that are related to reading
// and writing Claw struct types from wire encoding.  THIS FILE IS PUBLIC ONLY OUT OF
// NECCESSITY AND ANY USE IS NOT PROTECTED between any versions. Seriously, your code
// will break if you use this.
package structs

import (
	"fmt"
	"io"
	"math"
	"sort"
	"sync/atomic"
	"unsafe"

	"github.com/bearlytools/claw/clawc/internal/binary"
	"github.com/bearlytools/claw/clawc/internal/bits"
	"github.com/bearlytools/claw/clawc/internal/typedetect"
	"github.com/bearlytools/claw/clawc/languages/go/conversions"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/reflect/enums"
	"github.com/bearlytools/claw/clawc/languages/go/structs/header"
	"github.com/gostdlib/base/context"
)

const (
	// maxDataSize is the max number that can fit into the dataSize field, which is 40 bits.
	maxDataSize = 1099511627775
)

// Number represents all int, uint and float types.
type Number = typedetect.Number

// GenericHeader is the header of struct.
type GenericHeader = header.Generic

func NewGenericHeader() GenericHeader {
	return header.New()
}

// FieldState represents the decode state of a field in lazy decoding mode.
type FieldState = mapping.FieldState

const (
	// stateRaw indicates the field data exists in rawData but hasn't been decoded yet.
	stateRaw FieldState = iota
	// stateDecoded indicates the field has been decoded from rawData into the fields slice.
	stateDecoded
	// stateDirty indicates the field has been modified and differs from rawData.
	stateDirty
)

// fieldOffset maps a field number to its location in the raw byte data.
// fieldOffset is an alias for mapping.FieldOffset to allow pooling.
type fieldOffset = mapping.FieldOffset

// StructField holds a struct field entry.
type StructField = mapping.StructField

// Struct is the basic type for holding a set of values. In claw format, every variable
// must be contained in a Struct.
type Struct struct {
	inList bool

	header GenericHeader
	fields []StructField
	excess []byte

	// mapping holds our Mapping object that allows us to understand what field number holds what value type.
	mapping *mapping.Map

	parent *Struct

	// structTotal is the total size of this struct in bytes, including header.
	structTotal atomic.Int64

	// isSetEnabled indicates if we track which fields were explicitly set.
	// When enabled, IsSet bitfield entries are appended to the struct during encoding.
	isSetEnabled bool
	// isSetBits tracks which fields have been explicitly set via Set* methods.
	// Each byte tracks 7 fields (bit 7 is continuation flag in encoding).
	// nil when isSetEnabled is false. Allocated from diffBuffers pool.
	isSetBits []byte

	// Lazy decoding support fields.
	// rawData holds the complete raw binary data for this struct (including header).
	// This is populated during Unmarshal and used for lazy field decoding.
	// Allocated from diffBuffers pool and returned on Reset/Recycle.
	rawData []byte
	// offsets maps field numbers to their byte offsets in rawData.
	// Sorted by fieldNum for binary search.
	offsets []fieldOffset
	// fieldStates tracks the decode state of each field (raw, decoded, or dirty).
	// Index corresponds to field number.
	fieldStates []FieldState
	// modified is true if any field has been modified via Set* or Delete* operations.
	// When false and rawData is non-nil, Marshal can write rawData directly.
	modified bool
	// decoding is true during the Unmarshal process. This prevents the lazy decode
	// size adjustment logic from being applied during initial decoding.
	decoding bool
}

// New creates a NewStruct that is used to create a *Struct for a specific data type.
func New(fieldNum uint16, dataMap *mapping.Map) *Struct {
	if dataMap == nil {
		panic("dataMap must not be nil")
	}

	ctx := context.Background()
	s := structPool.Get(ctx)
	s.structTotal.Store(0) // Reset in case struct was reused from pool
	s.header.SetFieldNum(fieldNum)
	s.header.SetFieldType(field.FTStruct)
	s.header.SetFinal40(0) // Reset header size
	s.mapping = dataMap

	if dataMap.StructFieldsPool != nil {
		s.fields = *dataMap.StructFieldsPool.Get(ctx)
	} else {
		s.fields = make([]StructField, len(dataMap.Fields))
	}

	XXXAddToTotal(s, 8) // the header
	return s
}

// NewFromReader creates a new Struct from data we read in.
func NewFromReader(r io.Reader, maps *mapping.Map) (*Struct, error) {
	s := New(0, maps)

	if _, err := s.Unmarshal(r); err != nil {
		return nil, err
	}
	return s, nil
}

// XXXSetIsSetEnabled enables IsSet tracking for this Struct. When enabled, bitfield
// entries tracking which fields were set are appended to the struct during encoding.
// As with all XXXFunName, this is meant to be used internally. Using this otherwise
// can have bad effects and there is no compatibility promise around it.
func (s *Struct) XXXSetIsSetEnabled() {
	s.isSetEnabled = true
	// Allocate bits: we need ceil(numFields / 7) bytes
	// (7 because bit 7 is used as continuation flag in encoding)
	numBytes := (len(s.mapping.Fields) + 6) / 7
	if numBytes == 0 {
		numBytes = 1
	}
	// Pad to 8-byte alignment for Claw wire format compatibility
	paddedSize := ((numBytes + 7) / 8) * 8
	s.isSetBits = diffBuffers.Get(context.Background(), paddedSize)[:paddedSize]
	clear(s.isSetBits) // Zero the buffer since pool may return dirty data
	// Add the padded bitfield size to total
	XXXAddToTotal(s, int64(paddedSize))
}

// markFieldSet marks a field as having been explicitly set.
func (s *Struct) markFieldSet(fieldNum uint16) {
	if !s.isSetEnabled || s.isSetBits == nil {
		return
	}
	byteIdx := int(fieldNum) / 7
	bitIdx := uint(fieldNum) % 7
	if byteIdx < len(s.isSetBits) {
		s.isSetBits[byteIdx] |= (1 << bitIdx)
	}
}

// isSetBitSize returns the size in bytes of the IsSet bitfield entries (padded to 8-byte alignment).
// Returns 0 if isSetEnabled is false.
func (s *Struct) isSetBitSize() int64 {
	if !s.isSetEnabled || s.isSetBits == nil {
		return 0
	}
	return int64(len(s.isSetBits)) // Already padded during allocation
}

// NewFrom creates a new Struct that represents the same Struct type.
func (s *Struct) NewFrom() *Struct {
	ctx := context.Background()
	h := GenericHeader(make([]byte, 8))
	h.SetFieldType(field.FTStruct)

	var fields []StructField
	if s.mapping.StructFieldsPool != nil {
		fields = *s.mapping.StructFieldsPool.Get(ctx)
	} else {
		fields = make([]StructField, len(s.mapping.Fields))
	}

	n := &Struct{
		header:  h,
		mapping: s.mapping,
		fields:  fields,
	}
	XXXAddToTotal(n, 8) // Use XXXAddToTotal to also update header
	// Propagate isSetEnabled if the parent has it enabled
	if s.isSetEnabled {
		n.XXXSetIsSetEnabled()
	}
	return n
}

// Map returns the mapping.Map associated with this Struct.
func (s *Struct) Map() *mapping.Map {
	return s.mapping
}

// Recycle returns the Struct to a pool. This will automatically call Reset().
func (s *Struct) Recycle(ctx context.Context) {
	// This is what calls Reset() as it implements Resetter.
	structPool.Put(context.Background(), s)
}

// Reset resets the Struct to its initial state. This implements sync.Resetter.
func (s *Struct) Reset() {
	ctx := context.Background()

	// Recycle field resources before clearing
	s.recycleFields(ctx)

	// Reset header values but keep the underlying []byte
	s.header.SetFieldNum(0)
	s.header.SetFieldType(0)
	s.header.SetFinal40(0)

	// Return slices to the mapping's pools before clearing mapping
	if s.mapping != nil {
		if s.fields != nil {
			clear(s.fields)
			s.mapping.StructFieldsPool.Put(ctx, &s.fields)
		}
		if s.fieldStates != nil {
			clear(s.fieldStates)
			s.mapping.FieldStates.Put(ctx, &s.fieldStates)
		}
		if s.offsets != nil && s.mapping.OffsetsPool != nil {
			s.offsets = s.offsets[:0] // Reset length but keep capacity
			s.mapping.OffsetsPool.Put(ctx, &s.offsets)
		}
	}

	// Clear all other fields
	s.inList = false
	s.fields = nil
	s.fieldStates = nil
	s.excess = nil
	s.mapping = nil
	s.parent = nil
	s.structTotal.Store(0)
	s.isSetEnabled = false
	// Return pooled isSetBits buffer before clearing
	if s.isSetBits != nil {
		diffBuffers.Put(ctx, s.isSetBits)
		s.isSetBits = nil
	}

	// Return pooled rawData buffer before clearing
	if s.rawData != nil {
		diffBuffers.Put(ctx, s.rawData)
		s.rawData = nil
	}

	s.offsets = nil
	s.modified = false
	s.decoding = false
}

// recycleFields returns decoded field resources to their respective pools.
func (s *Struct) recycleFields(ctx context.Context) {
	if s.mapping == nil || s.fields == nil {
		return
	}

	for i, f := range s.fields {
		if f.Header == nil {
			continue
		}

		// Skip fields still in raw state (not decoded)
		if s.fieldStates != nil && s.fieldStates[i] == stateRaw {
			continue
		}

		desc := s.mapping.Fields[i]
		isDirty := s.fieldStates != nil && s.fieldStates[i] == stateDirty

		switch desc.Type {
		case field.FTStruct:
			if isDirty && f.Ptr != nil {
				sub := (*Struct)(f.Ptr)
				sub.Reset()
				structPool.Put(ctx, sub)
			}
		case field.FTListStructs:
			if isDirty && f.Ptr != nil {
				list := (*Structs)(f.Ptr)
				for _, sub := range list.data {
					sub.Reset()
					structPool.Put(ctx, sub)
				}
				list.Reset()
			}
		case field.FTListBools:
			if f.Ptr != nil {
				b := (*Bools)(f.Ptr)
				b.data = nil
				b.len = 0
				b.s = nil
				boolPool.Put(ctx, b)
			}
		case field.FTListBytes, field.FTListStrings:
			if f.Ptr != nil {
				b := (*Bytes)(f.Ptr)
				b.header = nil
				b.offsets = nil
				b.data = nil
				b.s = nil
				b.dataSize.Store(0)
				b.padding.Store(0)
				bytesPool.Put(ctx, b)
			}
		case field.FTListInt8:
			if f.Ptr != nil {
				n := (*Numbers[int8])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nInt8Pool.Put(ctx, n)
			}
		case field.FTListInt16:
			if f.Ptr != nil {
				n := (*Numbers[int16])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nInt16Pool.Put(ctx, n)
			}
		case field.FTListInt32:
			if f.Ptr != nil {
				n := (*Numbers[int32])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nInt32Pool.Put(ctx, n)
			}
		case field.FTListInt64:
			if f.Ptr != nil {
				n := (*Numbers[int64])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nInt64Pool.Put(ctx, n)
			}
		case field.FTListUint8:
			if f.Ptr != nil {
				n := (*Numbers[uint8])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nUint8Pool.Put(ctx, n)
			}
		case field.FTListUint16:
			if f.Ptr != nil {
				n := (*Numbers[uint16])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nUint16Pool.Put(ctx, n)
			}
		case field.FTListUint32:
			if f.Ptr != nil {
				n := (*Numbers[uint32])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nUint32Pool.Put(ctx, n)
			}
		case field.FTListUint64:
			if f.Ptr != nil {
				n := (*Numbers[uint64])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nUint64Pool.Put(ctx, n)
			}
		case field.FTListFloat32:
			if f.Ptr != nil {
				n := (*Numbers[float32])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nFloat32Pool.Put(ctx, n)
			}
		case field.FTListFloat64:
			if f.Ptr != nil {
				n := (*Numbers[float64])(f.Ptr)
				n.data = nil
				n.len = 0
				n.s = nil
				nFloat64Pool.Put(ctx, n)
			}
		}
	}
}

// Fields returns the list of StructFields.
func (s *Struct) Fields() []StructField {
	return s.fields
}

// findOffsetIndex returns the index into s.offsets for the given fieldNum,
// or -1 if the field is not present in the raw data.
// Uses binary search since offsets are sorted by fieldNum.
func (s *Struct) findOffsetIndex(fieldNum uint16) int {
	if s.offsets == nil {
		return -1
	}
	idx := sort.Search(len(s.offsets), func(i int) bool {
		return s.offsets[i].FieldNum >= fieldNum
	})
	if idx < len(s.offsets) && s.offsets[idx].FieldNum == fieldNum {
		return idx
	}
	return -1
}

// fieldExistsInRaw returns true if the field exists in the raw byte data.
func (s *Struct) fieldExistsInRaw(fieldNum uint16) bool {
	return s.findOffsetIndex(fieldNum) >= 0
}

// rawFieldSize returns the size in bytes of a field in the raw data,
// or 0 if the field doesn't exist in raw data.
func (s *Struct) rawFieldSize(fieldNum uint16) int {
	idx := s.findOffsetIndex(fieldNum)
	if idx < 0 {
		return 0
	}
	return int(s.offsets[idx].Size)
}

// markModified marks this struct and all parent structs as modified.
// This is called when any Set* or Delete* operation occurs.
func (s *Struct) markModified() {
	for ptr := s; ptr != nil; ptr = ptr.parent {
		ptr.modified = true
	}
}

// isLazyRaw returns true if lazy decode is active and the field is in raw state.
// Returns false during initial unmarshal (s.decoding == true) to prevent
// the lazy decode size adjustment logic from being applied.
func (s *Struct) isLazyRaw(fieldNum uint16) bool {
	return !s.decoding && s.fieldStates != nil && s.fieldStates[fieldNum] == stateRaw
}

// markFieldDirty marks the field as dirty and propagates modified flag.
// Should be called after modifying a field. Safe to call during decoding (no-op).
func (s *Struct) markFieldDirty(fieldNum uint16) {
	if s.decoding {
		return
	}
	if s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	s.markModified()
}

// getOrDecode returns the StructField for the given fieldNum, decoding from
// raw bytes if necessary. Returns nil if the field doesn't exist.
// This is the core of lazy decoding - fields are only decoded when accessed.
func (s *Struct) getOrDecode(fieldNum uint16) *StructField {
	// If no lazy decode infrastructure, fall back to direct access
	if s.fieldStates == nil {
		return &s.fields[fieldNum]
	}

	// Fast path: field already decoded or modified
	if s.fieldStates[fieldNum] != stateRaw {
		return &s.fields[fieldNum]
	}

	// Field is in raw state - check if it exists in raw data
	idx := s.findOffsetIndex(fieldNum)
	if idx < 0 {
		// Field not present in raw data
		return &s.fields[fieldNum]
	}

	// Decode from raw bytes
	offset := s.offsets[idx]
	data := s.rawData[8+offset.Offset : 8+offset.Offset+offset.Size]
	s.decodeFieldFromRaw(fieldNum, data)
	s.fieldStates[fieldNum] = stateDecoded

	return &s.fields[fieldNum]
}

// decodeFieldFromRaw decodes a single field from raw bytes into the fields slice.
// The data parameter should contain the complete field data including header.
func (s *Struct) decodeFieldFromRaw(fieldNum uint16, data []byte) {
	if len(data) < 8 {
		return
	}

	// Use O(1) function pointer dispatch
	lazyDecoders := s.mapping.LazyDecoders
	if lazyDecoders != nil && int(fieldNum) < len(lazyDecoders) && lazyDecoders[fieldNum] != nil {
		desc := s.mapping.Fields[fieldNum]
		lazyDecoders[fieldNum](unsafe.Pointer(s), fieldNum, data, desc)
		return
	}

	// Fallback to type switch for backward compatibility
	s.decodeFieldFromRawFallback(fieldNum, data)
}

// decodeFieldFromRawFallback uses type switch for backward compatibility.
func (s *Struct) decodeFieldFromRawFallback(fieldNum uint16, data []byte) {
	h := GenericHeader(data[:8])
	fieldType := field.Type(h.FieldType())
	f := &s.fields[fieldNum]

	switch fieldType {
	case field.FTBool:
		f.Header = data[:8]
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		f.Header = data[:8]
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		f.Header = data[:8]
		if len(data) >= 16 {
			v := data[8:16]
			f.Ptr = unsafe.Pointer(&v)
		}
	case field.FTString, field.FTBytes:
		f.Header = data[:8]
		size := h.Final40()
		if size > 0 && len(data) >= int(8+size) {
			b := data[8 : 8+size]
			f.Ptr = unsafe.Pointer(&b)
		}
	case field.FTStruct:
		s.decodeStructFieldFromRaw(fieldNum, data)
	case field.FTListBools:
		s.decodeListBoolFromRaw(fieldNum, data)
	case field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
		field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
		field.FTListFloat32, field.FTListFloat64:
		s.decodeListNumberFromRaw(fieldNum, data)
	case field.FTListBytes, field.FTListStrings:
		s.decodeListBytesFromRaw(fieldNum, data)
	case field.FTListStructs:
		s.decodeListStructsFromRaw(fieldNum, data)
	}
}

// decodeStructFieldFromRaw decodes a nested struct field from raw bytes.
func (s *Struct) decodeStructFieldFromRaw(fieldNum uint16, data []byte) {
	m := s.mapping.Fields[fieldNum].Mapping
	if m == nil {
		m = s.mapping // Self-referential
	}

	sub := New(fieldNum, m)
	// Propagate isSetEnabled to nested struct before unmarshaling
	if s.isSetEnabled {
		sub.XXXSetIsSetEnabled()
	}
	// Create a reader from the data and unmarshal
	r := readers.Get(context.Background())
	r.Reset(data)
	defer readers.Put(context.Background(), r)

	_, err := sub.Unmarshal(r)
	if err != nil {
		return
	}

	sub.parent = s
	f := &s.fields[fieldNum]
	f.Header = sub.header
	f.Ptr = unsafe.Pointer(sub)
}

// decodeListBoolFromRaw decodes a list of bools from raw bytes.
func (s *Struct) decodeListBoolFromRaw(fieldNum uint16, data []byte) {
	dataCopy := data
	h, ptr, err := NewBoolsFromBytes(&dataCopy, nil) // Don't add to total - already counted
	if err != nil {
		return
	}
	ptr.s = s
	f := &s.fields[fieldNum]
	f.Header = h
	f.Ptr = unsafe.Pointer(ptr)
}

// decodeListNumberFromRaw decodes a list of numbers from raw bytes.
func (s *Struct) decodeListNumberFromRaw(fieldNum uint16, data []byte) {
	m := s.mapping.Fields[fieldNum]
	f := &s.fields[fieldNum]
	f.Header = data[:8]

	dataCopy := data
	var uptr unsafe.Pointer

	switch m.Type {
	case field.FTListInt8:
		ptr, err := NewNumbersFromBytes[int8](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt16:
		ptr, err := NewNumbersFromBytes[int16](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt32:
		ptr, err := NewNumbersFromBytes[int32](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListInt64:
		ptr, err := NewNumbersFromBytes[int64](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint8:
		ptr, err := NewNumbersFromBytes[uint8](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint16:
		ptr, err := NewNumbersFromBytes[uint16](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint32:
		ptr, err := NewNumbersFromBytes[uint32](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListUint64:
		ptr, err := NewNumbersFromBytes[uint64](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListFloat32:
		ptr, err := NewNumbersFromBytes[float32](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	case field.FTListFloat64:
		ptr, err := NewNumbersFromBytes[float64](&dataCopy, nil)
		if err != nil {
			return
		}
		ptr.s = s
		uptr = unsafe.Pointer(ptr)
	}
	f.Ptr = uptr
}

// decodeListBytesFromRaw decodes a list of bytes from raw bytes.
func (s *Struct) decodeListBytesFromRaw(fieldNum uint16, data []byte) {
	dataCopy := data
	ptr, err := NewBytesFromBytes(&dataCopy, nil)
	if err != nil {
		return
	}
	ptr.s = s
	f := &s.fields[fieldNum]
	f.Header = ptr.header
	f.Ptr = unsafe.Pointer(ptr)
}

// decodeListStructsFromRaw decodes a list of structs from raw bytes.
func (s *Struct) decodeListStructsFromRaw(fieldNum uint16, data []byte) {
	m := s.mapping.Fields[fieldNum].Mapping
	dataCopy := data
	l, err := NewStructsFromBytesWithIsSet(&dataCopy, nil, m, s.isSetEnabled)
	if err != nil {
		return
	}
	l.s = s
	l.isSetEnabled = s.isSetEnabled
	f := &s.fields[fieldNum]
	f.Header = l.header
	f.Ptr = unsafe.Pointer(l)
}

// IsSet determines if our Struct has a field set or not. If the fieldNum is invalid,
// this simply returns false. When the IsSet file option is enabled, this checks the
// bitfield for accurate tracking. Otherwise, it falls back to checking if the field
// has a header (which may not detect zero-value fields due to compression).
func (s *Struct) IsSet(fieldNum uint16) bool {
	if int(fieldNum) >= len(s.mapping.Fields) {
		return false
	}

	// If IsSet tracking is enabled, check the bitfield
	if s.isSetEnabled && s.isSetBits != nil {
		byteIdx := int(fieldNum) / 7
		bitIdx := uint(fieldNum) % 7
		if byteIdx < len(s.isSetBits) {
			return (s.isSetBits[byteIdx] & (1 << bitIdx)) != 0
		}
		return false
	}

	// IsSet not enabled - fall back to checking if field has a header or exists in raw data
	// Check lazy decode raw data first
	if s.isLazyRaw(fieldNum) {
		return s.fieldExistsInRaw(fieldNum)
	}
	// Check decoded fields
	return s.fields[fieldNum].Header != nil
}

var boolMask = bits.Mask[uint64](24, 25)

// GetBool gets a bool value from field at fieldNum. This return an error if the field
// is not a bool or fieldNum is not a valid field number. If the field is not set, it
// returns false with no error.
func GetBool(s *Struct, fieldNum uint16) (bool, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return false, err
	}

	f := s.getOrDecode(fieldNum)
	// Return the zero value of a non-set field.
	if f.Header == nil {
		return false, nil
	}

	i := binary.Get[uint64](f.Header)
	if bits.GetValue[uint64, uint8](i, boolMask, 24) == 1 {
		return true, nil
	}
	return false, nil
}

func MustGetBool(s *Struct, fieldNum uint16) bool {
	b, err := GetBool(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

// SetBool sets a boolean value in field "fieldNum" to value "value".
func SetBool(s *Struct, fieldNum uint16, value bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}

	s.markFieldSet(fieldNum)

	f := s.fields[fieldNum]
	isFromRaw := s.fieldStates != nil && s.fieldStates[fieldNum] == stateRaw

	// Track if previous value was true/false for updating structTotal on transitions
	var prevWasTrue bool
	if f.Header != nil {
		prevWasTrue = f.Header.Final40() != 0
	}

	if f.Header == nil {
		f.Header = NewGenericHeader()
		f.Header.SetFieldNum(fieldNum)
		f.Header.SetFieldType(field.FTBool)
		// Zero-value compression: only add to total for non-zero values
		if !isFromRaw && value {
			XXXAddToTotal(s, 8)
		}
	} else if !isFromRaw {
		// Handle true/false transitions for zero-value compression
		if !prevWasTrue && value {
			// Transition from false to true: add 8 bytes
			XXXAddToTotal(s, 8)
		} else if prevWasTrue && !value {
			// Transition from true to false: subtract 8 bytes
			XXXAddToTotal(s, -8)
		}
	}

	n := conversions.BytesToNum[uint64](f.Header)
	*n = bits.SetBit(*n, 24, value)
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag
	if s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	s.markModified()

	return nil
}

func MustSetBool(s *Struct, fieldNum uint16, value bool) {
	err := SetBool(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteBool deletes a boolean and updates our storage total.
func DeleteBool(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}

	// Handle size adjustment when deleting from raw state
	if s.isLazyRaw(fieldNum) {
		if s.fieldExistsInRaw(fieldNum) {
			XXXAddToTotal(s, -8) // Remove the 8-byte bool field
		}
	} else if s.fields[fieldNum].Header != nil {
		XXXAddToTotal(s, -8)
	}

	s.fields[fieldNum].Header = nil

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

// GetNumber gets a number value at fieldNum.
func GetNumber[N Number](s *Struct, fieldNum uint16) (N, error) {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return 0, err
	}
	desc := s.mapping.Fields[fieldNum]

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return 0, fmt.Errorf("error getting field number %d: %w", fieldNum, err)
	}

	f := s.getOrDecode(fieldNum)
	if f.Header == nil {
		return 0, nil
	}

	if size < 64 {
		b := f.Header[3:8]
		if isFloat {
			i := binary.Get[uint32](b)
			return N(math.Float32frombits(uint32(i))), nil
		}
		return N(binary.Get[uint32](b)), nil
	}
	b := *(*[]byte)(f.Ptr)
	if isFloat {
		i := binary.Get[uint64](b)
		return N(math.Float64frombits(uint64(i))), nil
	}
	return N(binary.Get[uint64](b)), nil
}

func MustGetNumber[N Number](s *Struct, fieldNum uint16) N {
	n, err := GetNumber[N](s, fieldNum)
	if err != nil {
		panic(err)
	}
	return n
}

// SetNumber sets a number value in field "fieldNum" to value "value".
func SetNumber[N Number](s *Struct, fieldNum uint16, value N) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping.Fields[fieldNum]

	s.markFieldSet(fieldNum)

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error setting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum]
	isFromRaw := s.fieldStates != nil && s.fieldStates[fieldNum] == stateRaw

	// Zero-value compression: zero values are not encoded, so they don't contribute to total
	isZero := value == 0

	// Track if previous value was zero (for updating structTotal on transitions)
	var prevWasZero bool
	if f.Header == nil {
		prevWasZero = true // Unset field is treated as zero
	} else if size < 64 {
		prevWasZero = f.Header.Final40() == 0
	} else {
		// For 64-bit values, check if the data bytes are all zero
		if f.Ptr != nil {
			b := *(*[]byte)(f.Ptr)
			prevWasZero = true
			for _, v := range b {
				if v != 0 {
					prevWasZero = false
					break
				}
			}
		} else {
			prevWasZero = true
		}
	}

	// If the field isn't allocated, allocate space.
	if f.Header == nil {
		f.Header = NewGenericHeader()
		switch size < 64 {
		case true:
			// Only add to total if value is non-zero and not from raw
			if !isFromRaw && !isZero {
				XXXAddToTotal(s, 8)
			}
		case false:
			b := make([]byte, 8)
			f.Ptr = unsafe.Pointer(&b)
			// Only add to total if value is non-zero and not from raw
			if !isFromRaw && !isZero {
				XXXAddToTotal(s, 16)
			}
		default:
			panic("wtf")
		}
	} else if !isFromRaw {
		// Header already exists, handle zero/non-zero transitions
		if prevWasZero && !isZero {
			// Transitioning from zero to non-zero: add to total
			if size < 64 {
				XXXAddToTotal(s, 8)
			} else {
				XXXAddToTotal(s, 16)
			}
		} else if !prevWasZero && isZero {
			// Transitioning from non-zero to zero: subtract from total
			if size < 64 {
				XXXAddToTotal(s, -8)
			} else {
				XXXAddToTotal(s, -16)
			}
		}
	}

	// Its will store up to 2 uint64s that will be written. 1 is written if we can fit our value
	// in the header, 2 if we can't.
	ints := [2]uint64{}
	// Write our header information.
	ints[0] = bits.SetValue(fieldNum, ints[0], 0, 16)
	ints[0] = bits.SetValue(uint8(desc.Type), ints[0], 16, 24)

	// If the number is a float, convert it to the uint representation.
	if isFloat {
		switch size < 64 {
		case true:
			i := math.Float32bits(float32(value))
			ints[0] = bits.SetValue(i, ints[0], 24, 64)
			binary.Put(f.Header[:8], ints[0])
		case false:
			i := math.Float64bits(float64(value))
			ints[1] = i
			binary.Put(f.Header[:8], ints[0])
			d := (*[]byte)(f.Ptr)
			binary.Put(*d, ints[1])
		default:
			panic("wtf")
		}
	} else {
		// Now encode the Number.
		switch size < 64 {
		case true:
			ints[0] = bits.SetValue(uint32(value), ints[0], 24, 64)
			binary.Put(f.Header[:8], ints[0])
		case false:
			ints[1] = uint64(value)
			binary.Put(f.Header[:8], ints[0])
			d := (*[]byte)(f.Ptr)
			binary.Put(*d, ints[1])
		default:
			panic("wtf")
		}
	}
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag
	if s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	s.markModified()

	return nil
}

func MustSetNumber[N Number](s *Struct, fieldNum uint16, value N) {
	err := SetNumber[N](s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteNumber deletes the number and updates our storage total.
func DeleteNumber(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping.Fields[fieldNum]

	// Only adjust total if field actually exists
	fieldExists := false
	if s.isLazyRaw(fieldNum) {
		fieldExists = s.fieldExistsInRaw(fieldNum)
	} else {
		fieldExists = s.fields[fieldNum].Header != nil
	}

	if fieldExists {
		switch desc.Type {
		case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
			XXXAddToTotal(s, -8)
		case field.FTInt64, field.FTUint64, field.FTFloat64:
			XXXAddToTotal(s, -16)
		default:
			panic("wtf")
		}
	}

	f := s.fields[fieldNum]
	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

// GetBytes returns a field of bytes (also our string as well in []byte form). If the value was not
// set, this is returned as nil. If it was set, but empty, this will be []byte{}. It is UNSAFE to modify
// this.
func GetBytes(s *Struct, fieldNum uint16) (*[]byte, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return nil, err
	}

	f := s.getOrDecode(fieldNum)
	if f.Header == nil { // The zero value
		return nil, nil
	}

	if f.Ptr == nil { // Set, but value is empty
		return nil, nil
	}

	x := (*[]byte)(f.Ptr)
	return x, nil
}

func MustGetBytes(s *Struct, fieldNum uint16) *[]byte {
	b, err := GetBytes(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

// SetBytes sets a field of bytes (also our string as well in []byte form).
func SetBytes(s *Struct, fieldNum uint16, value []byte, isString bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}
	if len(value) == 0 {
		f := s.fields[fieldNum]
		if f.Header == nil && (s.fieldStates == nil || s.fieldStates[fieldNum] != stateRaw || !s.fieldExistsInRaw(fieldNum)) {
			// Field not set, nothing to do
			return nil
		}
		// Field is set, delete it
		return DeleteBytes(s, fieldNum)
	}

	if len(value) > maxDataSize {
		return fmt.Errorf("cannot set a String or Byte field to size > 1099511627775")
	}

	s.markFieldSet(fieldNum)

	f := s.fields[fieldNum]

	ftype := field.FTBytes
	if isString {
		ftype = field.FTString
	}

	// Handle size adjustment
	if f.Header == nil {
		// Field not currently allocated
		if s.isLazyRaw(fieldNum) && s.fieldExistsInRaw(fieldNum) {
			// Transitioning from raw - remove old size, add new size
			oldSize := s.rawFieldSize(fieldNum)
			newSize := 8 + SizeWithPadding(len(value))
			XXXAddToTotal(s, int64(newSize-oldSize))
		} else {
			// New field
			XXXAddToTotal(s, int64(8+SizeWithPadding(len(value))))
		}
		f.Header = NewGenericHeader()
	} else {
		// We need to remove our existing entry size total before applying our new data
		remove := 8 + SizeWithPadding(int(f.Header.Final40()))
		XXXAddToTotal(s, -remove)
		XXXAddToTotal(s, int64(8+SizeWithPadding(len(value))))
	}

	f.Header.SetFieldNum(fieldNum)
	f.Header.SetFieldType(ftype)
	f.Header.SetFinal40(uint64(len(value)))

	f.Ptr = unsafe.Pointer(&value)
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

func MustSetBytes(s *Struct, fieldNum uint16, value []byte, isString bool) {
	err := SetBytes(s, fieldNum, value, isString)
	if err != nil {
		panic(err)
	}
}

// DeleteBytes deletes a bytes field and updates our storage total.
func DeleteBytes(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}

	f := s.fields[fieldNum]

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Field in raw state
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else if f.Header != nil {
		// Field is decoded/dirty
		size := 8 + SizeWithPadding(int(f.Header.Final40()))
		XXXAddToTotal(s, -int64(size))
	} else {
		// Field not set, nothing to do
		return nil
	}

	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

// GetStruct returns a Struct field . If the value was not set, this is returned as nil. If it was set,
// but empty, this will be *Struct with no data.
func GetStruct(s *Struct, fieldNum uint16) (*Struct, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return nil, err
	}

	f := s.getOrDecode(fieldNum)
	if f.Header == nil { // The zero value
		return nil, nil
	}

	x := (*Struct)(f.Ptr)
	return x, nil
}

func MustGetStruct(s *Struct, fieldNum uint16) *Struct {
	s, err := GetStruct(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return s
}

// SetStruct sets a Struct field.
func SetStruct(s *Struct, fieldNum uint16, value *Struct) error {
	if s == nil {
		return fmt.Errorf("value cannot be added to a nil Struct")
	}
	if value == nil {
		return fmt.Errorf("value cannot be nil, to delete a Struct use DeleteStruct()")
	}
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return err
	}

	s.markFieldSet(fieldNum)

	if value.structTotal.Load() > maxDataSize {
		return fmt.Errorf("cannot set a Struct field to size > 1099511627775")
	}

	f := s.fields[fieldNum]

	// Propagate isSetEnabled BEFORE setting parent to avoid double-counting
	if s.isSetEnabled && !value.isSetEnabled {
		value.XXXSetIsSetEnabled()
	}

	value.parent = s
	value.header.SetFieldNum(fieldNum)

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Transitioning from raw to dirty
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else if f.Header != nil {
		// We need to remove our existing entry size total before applying our new data
		x := (*Struct)(f.Ptr)
		x.structTotal.Store(0)
		x.parent = nil
	}

	f.Header = value.header
	f.Ptr = unsafe.Pointer(value)
	XXXAddToTotal(s, value.structTotal.Load()) // Add child's size to parent
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

func MustSetStruct(s *Struct, fieldNum uint16, value *Struct) {
	err := SetStruct(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteStruct deletes a Struct field and updates our storage total.
func DeleteStruct(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return err
	}

	f := s.fields[fieldNum]

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Field in raw state
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		} else {
			return nil // Field not set
		}
	} else if f.Header == nil {
		return nil // Field not set
	} else if f.Ptr == nil {
		XXXAddToTotal(s, -8)
	} else {
		x := (*Struct)(f.Ptr)
		x.parent = nil
		s.structTotal.Add(-x.structTotal.Load())
	}

	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

// GetListBool returns a list of bools at fieldNum.
func GetListBool(s *Struct, fieldNum uint16) (*Bools, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBools); err != nil {
		return nil, err
	}

	f := s.getOrDecode(fieldNum)
	if f.Header == nil {
		return nil, nil
	}

	ptr := (*Bools)(f.Ptr)

	return ptr, nil
}

func MustGetListBool(s *Struct, fieldNum uint16) *Bools {
	b, err := GetListBool(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListBool(s *Struct, fieldNum uint16, value *Bools) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBools); err != nil {
		return err
	}

	s.markFieldSet(fieldNum)

	f := s.fields[fieldNum]

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else if f.Header != nil {
		ptr := (*Bools)(f.Ptr)
		XXXAddToTotal(s, -len(ptr.data))
	}

	f.Header = value.data[:8]
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f
	value.s = s
	XXXAddToTotal(s, len(value.data))

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

func MustSetListBool(s *Struct, fieldNum uint16, value *Bools) {
	err := SetListBool(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteListBools deletes a list of bools field and updates our storage total.
func DeleteListBools(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBools); err != nil {
		return err
	}

	f := s.fields[fieldNum]

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		} else {
			return nil
		}
	} else if f.Header == nil {
		return nil
	} else {
		ptr := (*Bools)(f.Ptr)
		XXXAddToTotal(s, -len(ptr.data))
	}

	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

// GetListNumber returns a list of numbers at fieldNum.
func GetListNumber[N Number](s *Struct, fieldNum uint16) (*Numbers[N], error) {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return nil, err
	}
	desc := s.mapping.Fields[fieldNum]

	f := s.getOrDecode(fieldNum)
	if f.Header == nil {
		return nil, nil
	}

	_, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return nil, fmt.Errorf("error getting field number %d: %w", fieldNum, err)
	}

	ptr := (*Numbers[N])(f.Ptr)

	return ptr, nil
}

func MustGetListNumber[N Number](s *Struct, fieldNum uint16) *Numbers[N] {
	b, err := GetListNumber[N](s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListNumber[N Number](s *Struct, fieldNum uint16, value *Numbers[N]) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping.Fields[fieldNum]

	_, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error setting field number %d: %w", fieldNum, err)
	}

	s.markFieldSet(fieldNum)

	f := s.fields[fieldNum]

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Transitioning from raw to dirty
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else if f.Header != nil { // We had a previous value stored.
		ptr := (*Numbers[N])(f.Ptr)
		XXXAddToTotal(s, -len(ptr.data))
	}

	f.Header = value.data[:8]
	f.Header.SetFieldNum(fieldNum)
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f
	value.s = s
	XXXAddToTotal(s, len(value.data))

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

func MustSetListNumber[N Number](s *Struct, fieldNum uint16, value *Numbers[N]) {
	err := SetListNumber(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteListNumber deletes a list of numbers field and updates our storage total.
func DeleteListNumber[N Number](s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.NumericListTypes...); err != nil {
		return err
	}

	desc := s.mapping.Fields[fieldNum]
	size, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error deleting field number %d: %w", fieldNum, err)
	}

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Transitioning from raw to dirty
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else {
		f := s.fields[fieldNum]
		if f.Header == nil {
			return nil
		}
		ptr := (*Numbers[N])(f.Ptr)
		ptr.s = nil

		reduceBy := int(f.Header.Final40()) + int(size) + 8
		XXXAddToTotal(s, -reduceBy)
	}

	f := s.fields[fieldNum]
	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

// GetListStruct returns a list of Structs at fieldNum.
func GetListStruct(s *Struct, fieldNum uint16) (*Structs, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return nil, err
	}

	f := s.getOrDecode(fieldNum)
	if f.Header == nil { // The zero value
		return nil, nil
	}

	x := (*Structs)(f.Ptr)
	return x, nil
}

func MustGetListStruct(s *Struct, fieldNum uint16) *Structs {
	l, err := GetListStruct(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return l
}

// SetListStructs deletes all existing values and puts in the passed value.
func SetListStructs(s *Struct, fieldNum uint16, value *Structs) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return err
	}

	s.markFieldSet(fieldNum)

	value.isSetEnabled = s.isSetEnabled
	for _, v := range value.data {
		// Propagate isSetEnabled BEFORE setting parent to avoid double-counting
		if s.isSetEnabled && !v.isSetEnabled {
			v.XXXSetIsSetEnabled()
		}
		v.parent = s
	}

	if value.Len() > maxDataSize {
		return fmt.Errorf("cannot have more than %d items in a list", maxDataSize)
	}

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Transitioning from raw to dirty
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else {
		// Use the normal delete path to clean up decoded data
		if err := DeleteListStructs(s, fieldNum); err != nil {
			return err
		}
	}

	XXXAddToTotal(s, value.size.Load())
	f := s.fields[fieldNum]
	f.Header = value.header
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

func MustSetListStruct(s *Struct, fieldNum uint16, value *Structs) {
	if err := SetListStructs(s, fieldNum, value); err != nil {
		panic(err)
	}
}

// AppendListStruct adds the values to the list of Structs at fieldNum. Existing items will be retained.
func AppendListStruct(s *Struct, fieldNum uint16, values ...*Struct) error {
	if len(values) == 0 {
		return fmt.Errorf("must add at least a single value")
	}
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return err
	}

	s.markFieldSet(fieldNum)

	// Use getOrDecode to ensure the field is decoded if it exists in raw data
	f := s.getOrDecode(fieldNum)

	// The list of structs hasn't been created yet, so create it.
	if f.Header == nil {
		fd := s.mapping.Fields[fieldNum]
		var l *Structs
		if fd.SelfReferential {
			l = NewStructs(s.mapping)
		} else {
			l = NewStructs(fd.Mapping)
		}
		f.Ptr = unsafe.Pointer(l)
		XXXAddToTotal(s, 8) // Add the header size of Structs to our parent Struct
	}

	l := (*Structs)(f.Ptr)
	l.s = s
	l.isSetEnabled = s.isSetEnabled

	if len(values)+l.Len() > maxDataSize {
		return fmt.Errorf("cannot have more than %d items in a list", maxDataSize)
	}

	for _, v := range values {
		if v == nil {
			return fmt.Errorf("cannot pass a nil *Struct")
		}
	}

	if err := l.Append(values...); err != nil {
		return err
	}
	l.header.SetFieldNum(fieldNum)
	l.header.SetFieldType(field.FTListStructs)
	f.Header = l.header

	f.Ptr = unsafe.Pointer(l)
	s.fields[fieldNum] = *f

	l.s = s

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

func MustAppendListStruct(s *Struct, fieldNum uint16, values ...*Struct) {
	err := AppendListStruct(s, fieldNum, values...)
	if err != nil {
		panic(err)
	}
}

// DeleteListStructs deletes a list of Structs field and updates our storage total.
func DeleteListStructs(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStructs); err != nil {
		return err
	}

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Transitioning from raw to dirty
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else {
		f := s.fields[fieldNum]
		if f.Header == nil {
			return nil
		}
		x := (*Structs)(f.Ptr)
		XXXAddToTotal(s, -x.size.Load())
	}

	f := s.fields[fieldNum]
	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

// GetListBytes returns a list of bytes at fieldNum.
func GetListBytes(s *Struct, fieldNum uint16) (*Bytes, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return nil, err
	}

	f := s.getOrDecode(fieldNum)
	if f.Header == nil {
		return nil, nil
	}

	ptr := (*Bytes)(f.Ptr)

	return ptr, nil
}

func MustGetListBytes(s *Struct, fieldNum uint16) *Bytes {
	b, err := GetListBytes(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListBytes(s *Struct, fieldNum uint16, value *Bytes) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return err
	}

	s.markFieldSet(fieldNum)

	value.s = s

	f := s.fields[fieldNum]

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Transitioning from raw to dirty
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
		// Add new value size
		XXXAddToTotal(s, value.dataSize.Load()+value.padding.Load()+8)
	} else if f.Header == nil {
		XXXAddToTotal(s, value.dataSize.Load()+value.padding.Load()+8)
	} else {
		ptr := (*Bytes)(f.Ptr)
		XXXAddToTotal(s, value.dataSize.Load()-ptr.dataSize.Load()+value.padding.Load()-ptr.padding.Load()+8)
	}

	value.header.SetFieldNum(fieldNum)
	f.Header = value.header
	f.Ptr = unsafe.Pointer(value)
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

func MustSetListBytes(s *Struct, fieldNum uint16, value *Bytes) {
	if err := SetListBytes(s, fieldNum, value); err != nil {
		panic(err)
	}
}

func MustSetListStrings(s *Struct, fieldNum uint16, value *Strings) {
	if err := SetListBytes(s, fieldNum, value.Bytes()); err != nil {
		panic(err)
	}
}

// DeleteListBytes deletes a list of bytes field and updates our storage total.
func DeleteListBytes(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return err
	}

	// Handle size adjustment based on current state
	// Skip lazy decode logic during initial unmarshal (s.decoding == true)
	if s.isLazyRaw(fieldNum) {
		// Transitioning from raw to dirty
		if s.fieldExistsInRaw(fieldNum) {
			oldSize := s.rawFieldSize(fieldNum)
			XXXAddToTotal(s, -int64(oldSize))
		}
	} else {
		f := s.fields[fieldNum]
		if f.Header == nil {
			return nil
		}
		ptr := (*Bytes)(f.Ptr)
		XXXAddToTotal(s, -(ptr.dataSize.Load() + ptr.padding.Load()))
	}

	f := s.fields[fieldNum]
	f.Header = nil
	f.Ptr = nil
	s.fields[fieldNum] = f

	// Mark as dirty and propagate modified flag (only after unmarshal is done)
	if !s.decoding && s.fieldStates != nil {
		s.fieldStates[fieldNum] = stateDirty
	}
	if !s.decoding {
		s.markModified()
	}

	return nil
}

type structer interface {
	Struct() *Struct
}

// SetField sets the field value at fieldNum to value. If value isn't valid for that field,
// this will panic.
func SetField(s *Struct, fieldNum uint16, value any) {
	if int(fieldNum) > len(s.fields) {
		panic(fmt.Sprintf("fieldNum %d is invalid", fieldNum))
	}

	switch t := s.mapping.Fields[int(fieldNum)].Type; t {
	case field.FTBool:
		v := value.(bool)
		MustSetBool(s, fieldNum, v)
	case field.FTInt8:
		v := value.(int8)
		MustSetNumber(s, fieldNum, v)
	case field.FTInt16:
		v := value.(int16)
		MustSetNumber(s, fieldNum, v)
	case field.FTInt32:
		v := value.(int32)
		MustSetNumber(s, fieldNum, v)
	case field.FTInt64:
		v := value.(int64)
		MustSetNumber(s, fieldNum, v)
	case field.FTUint8:
		switch v := value.(type) {
		case uint8:
			MustSetNumber(s, fieldNum, v)
		case enums.EnumImpl:
			if v.EnumSize != 8 {
				panic(fmt.Sprintf("setting a Uint8 field with a Enum that has size: %d", v.EnumSize))
			}
			MustSetNumber(s, fieldNum, uint8(v.EnumNumber))
		default:
			panic(fmt.Sprintf("setting a Uint8 field with a %T", value))
		}
	case field.FTUint16:
		switch v := value.(type) {
		case uint16:
			MustSetNumber(s, fieldNum, v)
		case enums.EnumImpl:
			if v.EnumSize != 16 {
				panic(fmt.Sprintf("setting a Uint16 field with a Enum that has size: %d", v.EnumSize))
			}
			MustSetNumber(s, fieldNum, v.EnumNumber)
		default:
			panic(fmt.Sprintf("setting a Uint16 field with a %T", value))
		}
	case field.FTUint32:
		v := value.(uint32)
		MustSetNumber(s, fieldNum, v)
	case field.FTUint64:
		v := value.(uint64)
		MustSetNumber(s, fieldNum, v)
	case field.FTFloat32:
		v := value.(float32)
		MustSetNumber(s, fieldNum, v)
	case field.FTFloat64:
		v := value.(float64)
		MustSetNumber(s, fieldNum, v)
	case field.FTBytes:
		v := value.([]byte)
		MustSetBytes(s, fieldNum, v, false)
	case field.FTString:
		v := value.(string)
		MustSetBytes(s, fieldNum, conversions.UnsafeGetBytes(v), true)
	case field.FTStruct:
		switch v := value.(type) {
		case structer:
			MustSetStruct(s, fieldNum, v.Struct())
		case *Struct:
			MustSetStruct(s, fieldNum, v)
		default:
			panic(fmt.Sprintf("tried to set a struct field with type %T", value))
		}
	case field.FTListBools:
		v := value.(*Bools)
		MustSetListBool(s, fieldNum, v)
	case field.FTListInt8:
		v := value.(*Numbers[int8])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListInt16:
		v := value.(*Numbers[int16])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListInt32:
		v := value.(*Numbers[int32])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListInt64:
		v := value.(*Numbers[int64])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint8:
		v := value.(*Numbers[uint8])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint16:
		v := value.(*Numbers[uint16])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint32:
		v := value.(*Numbers[uint32])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListUint64:
		v := value.(*Numbers[uint64])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListFloat32:
		v := value.(*Numbers[float32])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListFloat64:
		v := value.(*Numbers[float64])
		MustSetListNumber(s, fieldNum, v)
	case field.FTListBytes:
		v := value.(*Bytes)
		MustSetListBytes(s, fieldNum, v)
	case field.FTListStrings:
		v := value.(*Strings)
		MustSetListStrings(s, fieldNum, v)
	case field.FTListStructs:
		v := value.(*Structs)
		MustSetListStruct(s, fieldNum, v)
	default:
		panic(fmt.Sprintf("bug: unsupported type %T", t))
	}
}

// DeleteField will delete the field entry for fieldNum.
func DeleteField(s *Struct, fieldNum uint16) {
	if int(fieldNum) > len(s.fields) {
		panic(fmt.Sprintf("fieldNum %d is invalid", fieldNum))
	}

	switch t := s.mapping.Fields[int(fieldNum)].Type; t {
	case field.FTBool:
		DeleteBool(s, fieldNum)
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64, field.FTUint8,
		field.FTUint16, field.FTUint32, field.FTUint64, field.FTFloat32, field.FTFloat64:
		DeleteNumber(s, fieldNum)
	case field.FTBytes, field.FTString:
		DeleteBytes(s, fieldNum)
	case field.FTStruct:
		DeleteStruct(s, fieldNum)
	case field.FTListBools:
		DeleteListBools(s, fieldNum)
	case field.FTListInt8:
		DeleteListNumber[int8](s, fieldNum)
	case field.FTListInt16:
		DeleteListNumber[int16](s, fieldNum)
	case field.FTListInt32:
		DeleteListNumber[int32](s, fieldNum)
	case field.FTListInt64:
		DeleteListNumber[int64](s, fieldNum)
	case field.FTListUint8:
		DeleteListNumber[uint8](s, fieldNum)
	case field.FTListUint16:
		DeleteListNumber[uint16](s, fieldNum)
	case field.FTListUint32:
		DeleteListNumber[uint32](s, fieldNum)
	case field.FTListUint64:
		DeleteListNumber[uint64](s, fieldNum)
	case field.FTListFloat32:
		DeleteListNumber[float32](s, fieldNum)
	case field.FTListFloat64:
		DeleteListNumber[float64](s, fieldNum)
	case field.FTListBytes:
		DeleteListBytes(s, fieldNum)
	case field.FTListStrings:
		DeleteListBytes(s, fieldNum)
	case field.FTListStructs:
		DeleteListStructs(s, fieldNum)
	default:
		panic(fmt.Sprintf("bug: unsupported type %T", t))
	}
}

// XXXAddToTotal is used to increment the sizes of everything in a struct by some value.
func XXXAddToTotal[N int64 | int | uint | uint64](s *Struct, value N) {
	if s == nil {
		return
	}
	v := s.structTotal.Add(int64(value))
	s.header.SetFinal40(uint64(v))
	ptr := s.parent
	for {
		if ptr == nil {
			return
		}
		v := ptr.structTotal.Add(int64(value))
		ptr.header.SetFinal40(uint64(v))
		ptr = ptr.parent
	}
}

// XXXGetStructTotal returns the total size of the struct. Used by internals of generated packages.
func XXXGetStructTotal(s *Struct) int64 {
	if s == nil {
		return 0
	}
	return s.structTotal.Load()
}

// validateFieldNum will validate that the type is described in the mapping.Map,
// and if len(ftypes) != 0, that the ftype and mapping.Map[fieldNum].Type are the same.
func validateFieldNum(fieldNum uint16, maps *mapping.Map, ftypes ...field.Type) error {
	if int(fieldNum) >= len(maps.Fields) {
		return fmt.Errorf("fieldNum %d is >= the number of possible fields (%d)", fieldNum, len(maps.Fields))
	}
	if len(ftypes) == 0 {
		return nil
	}

	desc := maps.Fields[fieldNum]
	found := false
	for _, ftype := range ftypes {
		if desc.Type == ftype {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("fieldNum(%d) was %v, which was not valid", fieldNum, desc.Type)
	}
	return nil
}

func numberToDescCheck[N Number](desc *mapping.FieldDescr) (size uint8, isFloat bool, err error) {
	var t N
	typeSize := unsafe.Sizeof(t)

	// Determine characteristics using unsafe helpers
	isFloatType := typedetect.IsFloat[N]()
	isSigned := typedetect.IsSignedInteger[N]()

	// Map size and characteristics to field types and validate
	switch typeSize {
	case 1:
		if isSigned {
			switch desc.Type {
			case field.FTInt8, field.FTListInt8:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a int8 or []int8 type, was %v", desc.Type)
			}
		} else {
			switch desc.Type {
			case field.FTUint8, field.FTListUint8:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a uint8 or []uint8 type, was %v", desc.Type)
			}
		}
		size = 8
	case 2:
		if isSigned {
			switch desc.Type {
			case field.FTInt16, field.FTListInt16:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a int16 or []int16 type, was %v", desc.Type)
			}
		} else {
			switch desc.Type {
			case field.FTUint16, field.FTListUint16:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a uint16 or []uint16 type, was %v", desc.Type)
			}
		}
		size = 16
	case 4:
		if isFloatType {
			switch desc.Type {
			case field.FTFloat32, field.FTListFloat32:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a float32 or []float32 type, was %v", desc.Type)
			}
			size = 32
			isFloat = true
		} else if isSigned {
			switch desc.Type {
			case field.FTInt32, field.FTListInt32:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a int32 or []int32 type, was %v", desc.Type)
			}
			size = 32
		} else {
			switch desc.Type {
			case field.FTUint32, field.FTListUint32:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a uint32 or []uint32 type, was %v", desc.Type)
			}
			size = 32
		}
	case 8:
		if isFloatType {
			switch desc.Type {
			case field.FTFloat64, field.FTListFloat64:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a float64 or []float64 type, was %v", desc.Type)
			}
			size = 64
			isFloat = true
		} else if isSigned {
			switch desc.Type {
			case field.FTInt64, field.FTListInt64:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a int64 or []int64 type, was %v", desc.Type)
			}
			size = 64
		} else {
			switch desc.Type {
			case field.FTUint64, field.FTListUint64:
			default:
				return 0, false, fmt.Errorf("fieldNum is not a uint64 or []uint64 type, was %v", desc.Type)
			}
			size = 64
		}
	default:
		return 0, false, fmt.Errorf("passed a number value of %T that we do not support (size: %d bytes)", t, typeSize)
	}
	return size, isFloat, nil
}
