package segment

import (
	"encoding/binary"
	"math"
	"reflect"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

// GetListBools returns a Bools list for the given field, parsing existing data if present.
// Returns nil if the field is not set.
func GetListBools(s *Struct, fieldNum uint16) *Bools {
	// Check if we already have this list cached first (list may be set but not yet marshaled)
	if existing := s.GetList(fieldNum); existing != nil {
		return existing.(*Bools)
	}

	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Parse from segment data
	data := s.seg.data[offset : offset+size]
	if len(data) < HeaderSize {
		return nil
	}

	// Get count from header or calculate from data size
	_, _, final40 := DecodeHeader(data[0:HeaderSize])
	totalSize := int(final40)
	dataBytes := totalSize - HeaderSize

	// Unpack bits to bools
	items := make([]bool, 0)
	for byteIdx := 0; byteIdx < dataBytes; byteIdx++ {
		b := data[HeaderSize+byteIdx]
		for bitIdx := 0; bitIdx < 8; bitIdx++ {
			if (b & (1 << uint(bitIdx))) != 0 {
				items = append(items, true)
			} else {
				items = append(items, false)
			}
		}
	}

	b := &Bools{
		parent:   s,
		fieldNum: fieldNum,
		items:    items,
		dirty:    false,
	}
	s.SetList(fieldNum, b)
	return b
}

// GetListNumbers returns a Numbers list for the given field, parsing existing data if present.
// Returns nil if the field is not set.
func GetListNumbers[I Number](s *Struct, fieldNum uint16) *Numbers[I] {
	// Check if we already have this list cached first (list may be set but not yet marshaled)
	typeMismatch := false
	if existing := s.GetList(fieldNum); existing != nil {
		// Use safe type assertion - if types don't match (e.g., enum type vs primitive),
		// we fall through to parse from segment data
		if typedList, ok := existing.(*Numbers[I]); ok {
			return typedList
		}
		// Type mismatch - sync the list to segment first, then parse fresh
		// Don't store the parsed list back since it would corrupt the cache
		typeMismatch = true
		if syncer, ok := existing.(ListSyncer); ok {
			syncer.SyncToSegment()
		}
	}

	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Parse from segment data
	data := s.seg.data[offset : offset+size]
	if len(data) < HeaderSize {
		return nil
	}

	// Calculate item size
	var zero I
	itemSize := int(reflect.TypeOf(zero).Size())
	kind := reflect.TypeOf(zero).Kind()

	// Get data portion
	_, _, final40 := DecodeHeader(data[0:HeaderSize])
	totalSize := int(final40)
	dataLen := totalSize - HeaderSize
	// Remove padding
	dataLen = (dataLen / itemSize) * itemSize

	// Determine field type
	var ft field.Type
	switch kind {
	case reflect.Int8:
		ft = field.FTListInt8
	case reflect.Int16:
		ft = field.FTListInt16
	case reflect.Int32:
		ft = field.FTListInt32
	case reflect.Int64:
		ft = field.FTListInt64
	case reflect.Uint8:
		ft = field.FTListUint8
	case reflect.Uint16:
		ft = field.FTListUint16
	case reflect.Uint32:
		ft = field.FTListUint32
	case reflect.Uint64:
		ft = field.FTListUint64
	case reflect.Float32:
		ft = field.FTListFloat32
	case reflect.Float64:
		ft = field.FTListFloat64
	}

	n := &Numbers[I]{
		parent:   s,
		fieldNum: fieldNum,
		header:   make([]byte, HeaderSize),
		data:     make([]byte, dataLen),
		len:      dataLen / itemSize,
		dirty:    false,
	}
	EncodeHeader(n.header, fieldNum, ft, 0)
	copy(n.data, data[HeaderSize:HeaderSize+dataLen])
	// Only cache if there was no type mismatch, to avoid corrupting the cache
	// with a different generic type parameter
	if !typeMismatch {
		s.SetList(fieldNum, n)
	}
	return n
}

// GetListStrings returns a Strings list for the given field, parsing existing data if present.
// Returns nil if the field is not set.
func GetListStrings(s *Struct, fieldNum uint16) *Strings {
	// Check if we already have this list cached first (list may be set but not yet marshaled)
	if existing := s.GetList(fieldNum); existing != nil {
		return existing.(*Strings)
	}

	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Parse from segment data
	data := s.seg.data[offset : offset+size]
	if len(data) < HeaderSize {
		return nil
	}

	// Parse items: [header][len:4][data...][len:4][data...]...
	items := make([]string, 0)
	pos := HeaderSize
	for pos+4 <= len(data) {
		strLen := int(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4
		if strLen == 0 {
			items = append(items, "")
		} else if pos+strLen <= len(data) {
			items = append(items, string(data[pos:pos+strLen]))
			pos += strLen
		} else {
			break
		}
	}

	strs := &Strings{
		parent:   s,
		fieldNum: fieldNum,
		items:    items,
		dirty:    false,
	}
	s.SetList(fieldNum, strs)
	return strs
}

// GetListBytes returns a Bytes list for the given field, parsing existing data if present.
// Returns nil if the field is not set.
func GetListBytes(s *Struct, fieldNum uint16) *Bytes {
	// Check if we already have this list cached first (list may be set but not yet marshaled)
	if existing := s.GetList(fieldNum); existing != nil {
		return existing.(*Bytes)
	}

	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Parse from segment data
	data := s.seg.data[offset : offset+size]
	if len(data) < HeaderSize {
		return nil
	}

	// Parse items: [header][len:4][data...][len:4][data...]...
	items := make([][]byte, 0)
	pos := HeaderSize
	for pos+4 <= len(data) {
		itemLen := int(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4
		if itemLen == 0 {
			items = append(items, nil)
		} else if pos+itemLen <= len(data) {
			item := make([]byte, itemLen)
			copy(item, data[pos:pos+itemLen])
			items = append(items, item)
			pos += itemLen
		} else {
			break
		}
	}

	b := &Bytes{
		parent:   s,
		fieldNum: fieldNum,
		items:    items,
		dirty:    false,
	}
	s.SetList(fieldNum, b)
	return b
}

// GetListStructs returns a Structs list for the given field, parsing existing data if present.
// Returns nil if the field is not set.
// Note: NewStructs already handles parsing, so this is just an alias for consistency.
func GetListStructs(s *Struct, fieldNum uint16, m *mapping.Map) *Structs {
	// Check if we already have this list cached first (list may be set but not yet marshaled)
	if existing := s.GetList(fieldNum); existing != nil {
		return existing.(*Structs)
	}

	_, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// NewStructs already parses existing data
	return NewStructs(s, fieldNum, m)
}

// ClearField removes a field from the struct (sets to zero value).
// This also clears any cached list and removes dirty list entries for the field.
func ClearField(s *Struct, fieldNum uint16) {
	s.removeField(fieldNum)
	s.ClearListCache(fieldNum)
	// Remove any dirty list entries for this field
	s.removeDirtyListsForField(fieldNum)
}

// encodeInt8 encodes an int8 to bytes.
func EncodeInt8(v int8) []byte {
	return []byte{byte(v)}
}

// encodeUint8 encodes a uint8 to bytes.
func EncodeUint8(v uint8) []byte {
	return []byte{v}
}

// EncodeInt16 encodes an int16 to bytes.
func EncodeInt16(v int16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(v))
	return b
}

// EncodeUint16 encodes a uint16 to bytes.
func EncodeUint16(v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return b
}

// EncodeInt32 encodes an int32 to bytes.
func EncodeInt32(v int32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(v))
	return b
}

// EncodeUint32 encodes a uint32 to bytes.
func EncodeUint32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

// EncodeInt64 encodes an int64 to bytes.
func EncodeInt64(v int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(v))
	return b
}

// EncodeUint64 encodes a uint64 to bytes.
func EncodeUint64(v uint64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	return b
}

// EncodeFloat32 encodes a float32 to bytes.
func EncodeFloat32(v float32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, math.Float32bits(v))
	return b
}

// EncodeFloat64 encodes a float64 to bytes.
func EncodeFloat64(v float64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, math.Float64bits(v))
	return b
}

// SetListNumbersRaw sets a numeric list field directly from raw item bytes.
// This bypasses the typed list machinery to avoid type parameter issues with enum types.
// The ft parameter specifies the list field type (e.g., field.FTListUint8).
// The itemData contains the raw bytes for all items (without header).
func SetListNumbersRaw(s *Struct, fieldNum uint16, ft field.Type, itemData []byte) {
	if len(itemData) == 0 {
		// Empty list - remove the field
		s.removeField(fieldNum)
		s.ClearListCache(fieldNum)
		return
	}

	// Build field data: header + item data
	totalSize := HeaderSize + len(itemData)
	data := make([]byte, totalSize)
	EncodeHeader(data[:HeaderSize], fieldNum, ft, uint64(totalSize))
	copy(data[HeaderSize:], itemData)

	// Insert directly into segment
	s.insertField(fieldNum, data)

	// Clear list cache so next access parses fresh from segment
	s.ClearListCache(fieldNum)
}
