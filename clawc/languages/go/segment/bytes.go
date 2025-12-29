package segment

import (
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

// paddingNeeded returns the number of padding bytes needed to align to 8 bytes.
func paddingNeeded(size int) int {
	if size <= 0 {
		return 0
	}
	leftOver := size % 8
	if leftOver == 0 {
		return 0
	}
	return 8 - leftOver
}

// sizeWithPadding returns the size including padding for 8-byte alignment.
func sizeWithPadding(size int) int {
	return size + paddingNeeded(size)
}

// SetString sets a string field.
func SetString(s *Struct, fieldNum uint16, value string) {
	SetBytes(s, fieldNum, []byte(value))
}

// SetBytes sets a bytes field.
func SetBytes(s *Struct, fieldNum uint16, value []byte) {
	if len(value) == 0 {
		// Sparse encoding: remove empty values
		s.removeField(fieldNum)
		return
	}

	// Calculate total size: header + data + padding
	dataLen := len(value)
	padding := paddingNeeded(dataLen)
	totalSize := HeaderSize + dataLen + padding

	// Create the field data
	data := make([]byte, totalSize)

	// Write header with data size in Final40
	EncodeHeader(data[0:8], fieldNum, field.FTBytes, uint64(dataLen))

	// Copy data
	copy(data[8:8+dataLen], value)

	// Padding bytes are already zero from make()

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)
}

// SetStringAsBytes sets a string field using the FTString type instead of FTBytes.
func SetStringAsBytes(s *Struct, fieldNum uint16, value string) {
	if len(value) == 0 {
		s.removeField(fieldNum)
		return
	}

	dataLen := len(value)
	padding := paddingNeeded(dataLen)
	totalSize := HeaderSize + dataLen + padding

	data := make([]byte, totalSize)
	EncodeHeader(data[0:8], fieldNum, field.FTString, uint64(dataLen))
	copy(data[8:8+dataLen], value)

	s.insertField(fieldNum, data)
	s.markFieldSet(fieldNum)
}

// GetString gets a string field value.
func GetString(s *Struct, fieldNum uint16) string {
	b := GetBytes(s, fieldNum)
	if b == nil {
		return ""
	}
	return string(b)
}

// GetBytes gets a bytes field value.
// The returned slice is a view into the segment data and should not be modified.
func GetBytes(s *Struct, fieldNum uint16) []byte {
	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Get the data length from the header
	_, _, final40 := DecodeHeader(s.seg.data[offset : offset+HeaderSize])
	dataLen := int(final40)

	if dataLen == 0 {
		return nil
	}

	// Return the data (without padding)
	return s.seg.data[offset+HeaderSize : offset+HeaderSize+dataLen]
}

// GetBytesCopy gets a bytes field value as a copy.
func GetBytesCopy(s *Struct, fieldNum uint16) []byte {
	b := GetBytes(s, fieldNum)
	if b == nil {
		return nil
	}
	result := make([]byte, len(b))
	copy(result, b)
	return result
}

// SetNestedStruct sets a nested struct field.
// The child struct is embedded directly in the parent's segment.
func SetNestedStruct(parent *Struct, fieldNum uint16, child *Struct) {
	if child == nil {
		parent.removeField(fieldNum)
		return
	}

	// Sync any dirty lists in the child BEFORE checking size.
	// This ensures nested data (lists, sub-structs) is written to the segment
	// so we get an accurate size check.
	child.syncDirtyLists()

	if child.seg.Len() <= HeaderSize {
		// Empty struct: remove the field
		parent.removeField(fieldNum)
		return
	}

	// Create field data: our header + child's data (without child's root header)
	childData := child.seg.data[HeaderSize:] // Skip child's root header
	childSize := len(childData) + HeaderSize // Our header + child data

	data := make([]byte, childSize)

	// Write our header with the child's total size
	EncodeHeader(data[0:8], fieldNum, field.FTStruct, uint64(childSize))

	// Copy child data
	copy(data[8:], childData)

	parent.insertField(fieldNum, data)
	parent.markFieldSet(fieldNum)
}

// GetNestedStruct gets a nested struct field.
// This creates a new Struct that views the nested data in the parent's segment.
// Note: The returned struct shares the parent's segment data.
func GetNestedStruct(parent *Struct, fieldNum uint16, childMapping *mapping.Map) *Struct {
	offset, size := parent.FieldOffset(fieldNum)
	if size == 0 {
		return nil
	}

	// Create a new Struct that views this portion of the segment
	child := &Struct{
		seg:        &Segment{data: parent.seg.data[offset : offset+size]},
		mapping:    childMapping,
		fieldIndex: make([]fieldEntry, len(childMapping.Fields)),
	}

	// Parse the child's fields to populate fieldIndex
	parseFieldIndex(child)

	return child
}

// parseFieldIndex parses the segment data to populate the field index.
// This is used when reading nested structs from wire format.
func parseFieldIndex(s *Struct) {
	data := s.seg.data
	if len(data) <= HeaderSize {
		return
	}

	// Start parsing after the struct header
	pos := HeaderSize
	for pos+HeaderSize <= len(data) {
		fieldNum, fieldType, final40 := DecodeHeader(data[pos : pos+HeaderSize])

		// Calculate total field size based on type
		var fieldSize int
		switch fieldType {
		case field.FTBool, field.FTInt8, field.FTInt16, field.FTInt32,
			field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
			// Scalars that fit in header: just the header
			fieldSize = HeaderSize
		case field.FTInt64, field.FTUint64, field.FTFloat64:
			// 64-bit values: header + 8 bytes data
			fieldSize = HeaderSize + 8
		case field.FTString, field.FTBytes:
			// Variable length: header + data + padding
			dataLen := int(final40)
			padding := paddingNeeded(dataLen)
			fieldSize = HeaderSize + dataLen + padding
		case field.FTStruct:
			// Struct: final40 is total size including header
			fieldSize = int(final40)
		default:
			// Lists and other types: final40 is total size including header
			if field.IsListType(fieldType) {
				fieldSize = int(final40)
			} else {
				// Unknown type, skip using final40 as size
				fieldSize = int(final40)
				if fieldSize < HeaderSize {
					fieldSize = HeaderSize
				}
			}
		}

		// Record field position if within bounds
		if int(fieldNum) < len(s.fieldIndex) {
			s.fieldIndex[fieldNum] = fieldEntry{
				offset: uint32(pos),
				size:   uint32(fieldSize),
				isSet:  true,
			}
		}

		pos += fieldSize
	}
}
