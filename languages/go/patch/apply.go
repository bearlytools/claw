package patch

import (
	"bytes"
	"fmt"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/segment"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

const (
	// MaxPatchOps is the maximum number of operations allowed in a single patch.
	// This prevents denial-of-service from malicious/corrupt patches.
	MaxPatchOps = 10000

	// MaxPatchNestingDepth is the maximum nesting depth for STRUCT_PATCH operations.
	// This prevents stack overflow from deeply nested patches.
	MaxPatchNestingDepth = 100
)

// Apply applies the patch to 'base', modifying it in place.
// Returns error if patch cannot be applied.
func Apply[T ClawStruct](base T, p msgs.Patch) error {
	if p.Version() != PatchVersion {
		return fmt.Errorf("unsupported patch version: %d (expected %d)", p.Version(), PatchVersion)
	}

	opsLen := p.OpsLen()
	if opsLen > MaxPatchOps {
		return fmt.Errorf("patch has %d operations, exceeds maximum of %d", opsLen, MaxPatchOps)
	}

	s := base.XXXGetStruct()
	m := s.Mapping()

	for i := 0; i < opsLen; i++ {
		op := p.OpsGet(i)
		if err := applyOpWithDepth(s, m, op, 0); err != nil {
			return err
		}
	}

	return nil
}

// applyOpWithDepth applies a single operation to the struct, tracking nesting depth.
func applyOpWithDepth(s *segment.Struct, m *mapping.Map, op msgs.Op, depth int) error {
	fieldNum := op.FieldNum()
	opType := op.Type()
	data := op.Data()

	// Get field descriptor - may be nil for unknown fields
	var fd *mapping.FieldDescr
	if int(fieldNum) < len(m.Fields) {
		fd = m.Fields[fieldNum]
	}

	switch opType {
	case msgs.Set:
		return applySet(s, fd, fieldNum, data)
	case msgs.Clear:
		return applyClear(s, fd, fieldNum)
	case msgs.StructPatch:
		return applyStructPatch(s, fd, fieldNum, data, depth)
	case msgs.ListReplace:
		return applyListReplace(s, fd, fieldNum, data)
	case msgs.ListSet:
		return applyListSet(s, fd, fieldNum, op.Index(), data)
	case msgs.ListInsert:
		return applyListInsert(s, fd, fieldNum, op.Index(), data)
	case msgs.ListRemove:
		return applyListRemove(s, fd, fieldNum, op.Index())
	case msgs.ListStructPatch:
		return applyListStructPatch(s, fd, fieldNum, op.Index(), data, depth)
	default:
		// Unknown operation type, skip for forward compatibility
		return nil
	}
}

func applySet(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte) error {
	if fd == nil {
		// Unknown field - store as raw data for forward compatibility
		// Note: segment doesn't support unknown fields yet
		return nil
	}

	switch fd.Type {
	case field.FTBool:
		if len(data) < 1 {
			return fmt.Errorf("bool data too short")
		}
		segment.SetBool(s, fieldNum, data[0] != 0)
	case field.FTInt8:
		if len(data) < 1 {
			return fmt.Errorf("int8 data too short")
		}
		segment.SetInt8(s, fieldNum, int8(data[0]))
	case field.FTUint8:
		if len(data) < 1 {
			return fmt.Errorf("uint8 data too short")
		}
		segment.SetUint8(s, fieldNum, data[0])
	case field.FTInt16:
		if len(data) < 2 {
			return fmt.Errorf("int16 data too short")
		}
		segment.SetInt16(s, fieldNum, decodeInt16(data))
	case field.FTUint16:
		if len(data) < 2 {
			return fmt.Errorf("uint16 data too short")
		}
		segment.SetUint16(s, fieldNum, decodeUint16(data))
	case field.FTInt32:
		if len(data) < 4 {
			return fmt.Errorf("int32 data too short")
		}
		segment.SetInt32(s, fieldNum, decodeInt32(data))
	case field.FTUint32:
		if len(data) < 4 {
			return fmt.Errorf("uint32 data too short")
		}
		segment.SetUint32(s, fieldNum, decodeUint32(data))
	case field.FTInt64:
		if len(data) < 8 {
			return fmt.Errorf("int64 data too short")
		}
		segment.SetInt64(s, fieldNum, decodeInt64(data))
	case field.FTUint64:
		if len(data) < 8 {
			return fmt.Errorf("uint64 data too short")
		}
		segment.SetUint64(s, fieldNum, decodeUint64(data))
	case field.FTFloat32:
		if len(data) < 4 {
			return fmt.Errorf("float32 data too short")
		}
		segment.SetFloat32(s, fieldNum, decodeFloat32(data))
	case field.FTFloat64:
		if len(data) < 8 {
			return fmt.Errorf("float64 data too short")
		}
		segment.SetFloat64(s, fieldNum, decodeFloat64(data))
	case field.FTString:
		segment.SetStringAsBytes(s, fieldNum, string(data))
	case field.FTBytes:
		segment.SetBytes(s, fieldNum, data)
	case field.FTStruct:
		// For SET on struct, unmarshal the full struct
		subStruct := segment.New(fd.Mapping)
		if err := subStruct.Unmarshal(data); err != nil {
			return fmt.Errorf("unmarshal nested struct: %w", err)
		}
		segment.SetNestedStruct(s, fieldNum, subStruct)
	default:
		return fmt.Errorf("SET not supported for field type: %v", fd.Type)
	}

	return nil
}

func applyClear(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16) error {
	if fd == nil {
		// Unknown field - nothing to clear
		return nil
	}

	// For all types, use ClearField which removes the field from the segment
	segment.ClearField(s, fieldNum)
	return nil
}

func applyStructPatch(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte, depth int) error {
	if depth >= MaxPatchNestingDepth {
		return fmt.Errorf("patch nesting depth %d exceeds maximum of %d", depth, MaxPatchNestingDepth)
	}

	if fd == nil || fd.Type != field.FTStruct {
		return fmt.Errorf("STRUCT_PATCH requires struct field")
	}

	// Get the existing nested struct
	subStruct := segment.GetNestedStruct(s, fieldNum, fd.Mapping)
	if subStruct == nil {
		// Create new struct if doesn't exist
		subStruct = segment.New(fd.Mapping)
		segment.SetNestedStruct(s, fieldNum, subStruct)
	}

	// Unmarshal the patch
	subPatch := msgs.NewPatch(nil)
	if err := subPatch.Unmarshal(data); err != nil {
		return fmt.Errorf("unmarshal sub-patch: %w", err)
	}

	// Apply each operation to the nested struct
	opsLen := subPatch.OpsLen()
	for i := 0; i < opsLen; i++ {
		op := subPatch.OpsGet(i)
		if err := applyOpWithDepth(subStruct, fd.Mapping, op, depth+1); err != nil {
			return err
		}
	}

	return nil
}

func applyListReplace(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte) error {
	if fd == nil {
		return nil // Unknown field, skip
	}

	// Empty data means clear the list
	if len(data) == 0 {
		return applyClear(s, fd, fieldNum)
	}

	switch fd.Type {
	case field.FTListBools:
		return applyListReplaceBools(s, fieldNum, data)
	case field.FTListInt8, field.FTListUint8, field.FTListInt16, field.FTListUint16,
		field.FTListInt32, field.FTListUint32, field.FTListFloat32,
		field.FTListInt64, field.FTListUint64, field.FTListFloat64:
		return applyListReplaceNumbers(s, fieldNum, data, fd.Type)
	case field.FTListBytes, field.FTListStrings:
		return applyListReplaceBytes(s, fieldNum, data, fd.Type)
	case field.FTListStructs:
		return applyListReplaceStructs(s, fd, fieldNum, data)
	default:
		return fmt.Errorf("LIST_REPLACE not supported for field type: %v", fd.Type)
	}
}

func applyListReplaceBools(s *segment.Struct, fieldNum uint16, data []byte) error {
	bools := segment.NewBools(s, fieldNum)
	vals := make([]bool, len(data))
	for i, b := range data {
		vals[i] = b != 0
	}
	bools.SetAll(vals)
	return nil
}

func applyListReplaceNumbers(s *segment.Struct, fieldNum uint16, data []byte, ft field.Type) error {
	// Use raw setter to avoid type parameter issues with enum types
	segment.SetListNumbersRaw(s, fieldNum, ft, data)
	return nil
}

func applyListReplaceBytes(s *segment.Struct, fieldNum uint16, data []byte, ft field.Type) error {
	// Data format: [count:4][len1:4][data1...][len2:4][data2...]...
	if len(data) < 4 {
		return fmt.Errorf("bytes list data too short")
	}
	count := int(decodeUint32(data[:4]))
	offset := 4
	items := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		if offset+4 > len(data) {
			return fmt.Errorf("bytes list truncated at item %d", i)
		}
		itemLen := int(decodeUint32(data[offset:]))
		offset += 4
		if offset+itemLen > len(data) {
			return fmt.Errorf("bytes list item %d data truncated", i)
		}
		item := make([]byte, itemLen)
		copy(item, data[offset:offset+itemLen])
		items = append(items, item)
		offset += itemLen
	}

	if ft == field.FTListStrings {
		strs := segment.NewStrings(s, fieldNum)
		strItems := make([]string, len(items))
		for i, b := range items {
			strItems[i] = string(b)
		}
		strs.SetAll(strItems)
	} else {
		b := segment.NewBytes(s, fieldNum)
		b.SetAll(items)
	}
	return nil
}

func applyListReplaceStructs(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte) error {
	// Data format: [count:4][struct1...][struct2...]...
	if len(data) < 4 {
		return fmt.Errorf("struct list data too short")
	}
	count := int(decodeUint32(data[:4]))
	reader := bytes.NewReader(data[4:])

	list := segment.NewStructs(s, fieldNum, fd.Mapping)
	items := make([]*segment.Struct, 0, count)
	for i := 0; i < count; i++ {
		item := segment.New(fd.Mapping)
		if _, err := item.UnmarshalReader(reader); err != nil {
			return fmt.Errorf("unmarshal struct at index %d: %w", i, err)
		}
		items = append(items, item)
	}
	for _, item := range items {
		list.Append(item)
	}
	return nil
}

func applyListSet(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	if fd == nil {
		return nil // Unknown field, skip
	}
	if index < 0 {
		return fmt.Errorf("LIST_SET invalid index: %d", index)
	}

	switch fd.Type {
	case field.FTListBools:
		list := segment.GetListBools(s, fieldNum)
		if list == nil || int(index) >= list.Len() {
			return fmt.Errorf("LIST_SET index %d out of bounds", index)
		}
		if len(data) < 1 {
			return fmt.Errorf("LIST_SET bool data too short")
		}
		list.Set(int(index), data[0] != 0)
	case field.FTListInt8:
		return applyListSetNumber(s, fieldNum, index, data, 1, field.FTListInt8)
	case field.FTListUint8:
		return applyListSetNumber(s, fieldNum, index, data, 1, field.FTListUint8)
	case field.FTListInt16:
		return applyListSetNumber(s, fieldNum, index, data, 2, field.FTListInt16)
	case field.FTListUint16:
		return applyListSetNumber(s, fieldNum, index, data, 2, field.FTListUint16)
	case field.FTListInt32:
		return applyListSetNumber(s, fieldNum, index, data, 4, field.FTListInt32)
	case field.FTListUint32:
		return applyListSetNumber(s, fieldNum, index, data, 4, field.FTListUint32)
	case field.FTListFloat32:
		return applyListSetNumber(s, fieldNum, index, data, 4, field.FTListFloat32)
	case field.FTListInt64:
		return applyListSetNumber(s, fieldNum, index, data, 8, field.FTListInt64)
	case field.FTListUint64:
		return applyListSetNumber(s, fieldNum, index, data, 8, field.FTListUint64)
	case field.FTListFloat64:
		return applyListSetNumber(s, fieldNum, index, data, 8, field.FTListFloat64)
	case field.FTListBytes:
		list := segment.GetListBytes(s, fieldNum)
		if list == nil || int(index) >= list.Len() {
			return fmt.Errorf("LIST_SET index %d out of bounds", index)
		}
		list.Set(int(index), data)
	case field.FTListStrings:
		list := segment.GetListStrings(s, fieldNum)
		if list == nil || int(index) >= list.Len() {
			return fmt.Errorf("LIST_SET index %d out of bounds", index)
		}
		list.Set(int(index), string(data))
	case field.FTListStructs:
		list := segment.GetListStructs(s, fieldNum, fd.Mapping)
		if list == nil || int(index) >= list.Len() {
			return fmt.Errorf("LIST_SET index %d out of bounds", index)
		}
		item := segment.New(fd.Mapping)
		if err := item.Unmarshal(data); err != nil {
			return fmt.Errorf("unmarshal struct for LIST_SET: %w", err)
		}
		list.Set(int(index), item)
	default:
		return fmt.Errorf("LIST_SET not supported for field type: %v", fd.Type)
	}
	return nil
}

func applyListSetNumber(s *segment.Struct, fieldNum uint16, index int32, value []byte, sizeInBytes int, ft field.Type) error {
	// Get existing data from segment
	itemData, err := getNumberListItemData(s, fieldNum, sizeInBytes)
	if err != nil {
		return err
	}
	if int(index)*sizeInBytes >= len(itemData) {
		return fmt.Errorf("LIST_SET index %d out of bounds", index)
	}
	if len(value) < sizeInBytes {
		return fmt.Errorf("LIST_SET number data too short")
	}

	// Modify the byte(s) at the given index
	copy(itemData[int(index)*sizeInBytes:], value[:sizeInBytes])

	// Set the modified data
	segment.SetListNumbersRaw(s, fieldNum, ft, itemData)
	return nil
}

// getNumberListItemData extracts the item data bytes from a number list field.
// It syncs any dirty lists first to ensure segment data is up to date.
func getNumberListItemData(s *segment.Struct, fieldNum uint16, itemSize int) ([]byte, error) {
	// Sync any dirty lists for this field first
	s.SyncDirtyListsForField(fieldNum)

	offset, size := s.FieldOffset(fieldNum)
	if size == 0 {
		return nil, fmt.Errorf("field %d not found", fieldNum)
	}

	data := s.SegmentData()[offset : offset+size]
	if len(data) < segment.HeaderSize {
		return nil, fmt.Errorf("field %d data too short", fieldNum)
	}

	// Decode header to get total size
	_, _, final40 := segment.DecodeHeader(data[:segment.HeaderSize])
	totalSize := int(final40)
	dataLen := totalSize - segment.HeaderSize
	// Remove padding by truncating to multiple of item size
	dataLen = (dataLen / itemSize) * itemSize

	// Copy the item data
	itemData := make([]byte, dataLen)
	copy(itemData, data[segment.HeaderSize:segment.HeaderSize+dataLen])
	return itemData, nil
}

func applyListInsert(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	if fd == nil {
		return nil // Unknown field, skip
	}
	if index < 0 {
		return fmt.Errorf("LIST_INSERT invalid index: %d", index)
	}

	// For insert, we need to get the current list, convert to slice, insert, and create a new list
	switch fd.Type {
	case field.FTListBools:
		return applyListInsertBool(s, fieldNum, index, data)
	case field.FTListInt8:
		return applyListInsertNumber(s, fieldNum, index, data, 1, field.FTListInt8)
	case field.FTListUint8:
		return applyListInsertNumber(s, fieldNum, index, data, 1, field.FTListUint8)
	case field.FTListInt16:
		return applyListInsertNumber(s, fieldNum, index, data, 2, field.FTListInt16)
	case field.FTListUint16:
		return applyListInsertNumber(s, fieldNum, index, data, 2, field.FTListUint16)
	case field.FTListInt32:
		return applyListInsertNumber(s, fieldNum, index, data, 4, field.FTListInt32)
	case field.FTListUint32:
		return applyListInsertNumber(s, fieldNum, index, data, 4, field.FTListUint32)
	case field.FTListFloat32:
		return applyListInsertNumber(s, fieldNum, index, data, 4, field.FTListFloat32)
	case field.FTListInt64:
		return applyListInsertNumber(s, fieldNum, index, data, 8, field.FTListInt64)
	case field.FTListUint64:
		return applyListInsertNumber(s, fieldNum, index, data, 8, field.FTListUint64)
	case field.FTListFloat64:
		return applyListInsertNumber(s, fieldNum, index, data, 8, field.FTListFloat64)
	case field.FTListBytes:
		return applyListInsertBytes(s, fieldNum, index, data)
	case field.FTListStrings:
		return applyListInsertString(s, fieldNum, index, data)
	case field.FTListStructs:
		return applyListInsertStruct(s, fd, fieldNum, index, data)
	default:
		return fmt.Errorf("LIST_INSERT not supported for field type: %v", fd.Type)
	}
}

func applyListInsertBool(s *segment.Struct, fieldNum uint16, index int32, data []byte) error {
	list := segment.GetListBools(s, fieldNum)
	var existing []bool
	if list != nil {
		existing = list.Slice()
	}
	if int(index) > len(existing) {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, len(existing))
	}
	if len(data) < 1 {
		return fmt.Errorf("LIST_INSERT bool data too short")
	}
	newVal := data[0] != 0
	// Insert at index - single allocation, two copies
	newSlice := make([]bool, len(existing)+1)
	copy(newSlice, existing[:index])
	newSlice[index] = newVal
	copy(newSlice[index+1:], existing[index:])

	newList := segment.NewBools(s, fieldNum)
	newList.SetAll(newSlice)
	return nil
}

func applyListInsertNumber(s *segment.Struct, fieldNum uint16, index int32, value []byte, sizeInBytes int, ft field.Type) error {
	if len(value) < sizeInBytes {
		return fmt.Errorf("LIST_INSERT number data too short")
	}

	// Get existing item data (may be empty if list doesn't exist yet)
	var existing []byte
	offset, size := s.FieldOffset(fieldNum)
	if size > 0 {
		data := s.SegmentData()[offset : offset+size]
		if len(data) >= segment.HeaderSize {
			_, _, final40 := segment.DecodeHeader(data[:segment.HeaderSize])
			totalSize := int(final40)
			dataLen := totalSize - segment.HeaderSize
			dataLen = (dataLen / sizeInBytes) * sizeInBytes
			existing = make([]byte, dataLen)
			copy(existing, data[segment.HeaderSize:segment.HeaderSize+dataLen])
		}
	}

	// Calculate number of items
	numItems := len(existing) / sizeInBytes
	if int(index) > numItems {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, numItems)
	}

	// Insert new value at index
	insertOffset := int(index) * sizeInBytes
	newData := make([]byte, len(existing)+sizeInBytes)
	copy(newData, existing[:insertOffset])
	copy(newData[insertOffset:], value[:sizeInBytes])
	copy(newData[insertOffset+sizeInBytes:], existing[insertOffset:])

	segment.SetListNumbersRaw(s, fieldNum, ft, newData)
	return nil
}

func applyListInsertBytes(s *segment.Struct, fieldNum uint16, index int32, data []byte) error {
	list := segment.GetListBytes(s, fieldNum)
	var existing [][]byte
	if list != nil {
		existing = list.Slice()
	}
	if int(index) > len(existing) {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, len(existing))
	}
	// Insert at index - single allocation, two copies
	newSlice := make([][]byte, len(existing)+1)
	copy(newSlice, existing[:index])
	newSlice[index] = data
	copy(newSlice[index+1:], existing[index:])

	newList := segment.NewBytes(s, fieldNum)
	newList.SetAll(newSlice)
	return nil
}

func applyListInsertString(s *segment.Struct, fieldNum uint16, index int32, data []byte) error {
	list := segment.GetListStrings(s, fieldNum)
	var existing []string
	if list != nil {
		existing = list.Slice()
	}
	if int(index) > len(existing) {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, len(existing))
	}
	// Insert at index - single allocation, two copies
	newSlice := make([]string, len(existing)+1)
	copy(newSlice, existing[:index])
	newSlice[index] = string(data)
	copy(newSlice[index+1:], existing[index:])

	newList := segment.NewStrings(s, fieldNum)
	newList.SetAll(newSlice)
	return nil
}

func applyListInsertStruct(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	list := segment.GetListStructs(s, fieldNum, fd.Mapping)
	var existing []*segment.Struct
	if list != nil {
		existing = list.Slice()
	}
	if int(index) > len(existing) {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, len(existing))
	}
	item := segment.New(fd.Mapping)
	if err := item.Unmarshal(data); err != nil {
		return fmt.Errorf("unmarshal struct for LIST_INSERT: %w", err)
	}

	// Create new list with inserted item
	newList := segment.NewStructs(s, fieldNum, fd.Mapping)
	// Append items before index
	for i := 0; i < int(index); i++ {
		newList.Append(existing[i])
	}
	// Append new item
	newList.Append(item)
	// Append items after index
	for i := int(index); i < len(existing); i++ {
		newList.Append(existing[i])
	}
	return nil
}

func applyListRemove(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32) error {
	if fd == nil {
		return nil // Unknown field, skip
	}
	if index < 0 {
		return fmt.Errorf("LIST_REMOVE invalid index: %d", index)
	}

	switch fd.Type {
	case field.FTListBools:
		return applyListRemoveBool(s, fieldNum, index)
	case field.FTListInt8:
		return applyListRemoveNumber(s, fieldNum, index, 1, field.FTListInt8)
	case field.FTListUint8:
		return applyListRemoveNumber(s, fieldNum, index, 1, field.FTListUint8)
	case field.FTListInt16:
		return applyListRemoveNumber(s, fieldNum, index, 2, field.FTListInt16)
	case field.FTListUint16:
		return applyListRemoveNumber(s, fieldNum, index, 2, field.FTListUint16)
	case field.FTListInt32:
		return applyListRemoveNumber(s, fieldNum, index, 4, field.FTListInt32)
	case field.FTListUint32:
		return applyListRemoveNumber(s, fieldNum, index, 4, field.FTListUint32)
	case field.FTListFloat32:
		return applyListRemoveNumber(s, fieldNum, index, 4, field.FTListFloat32)
	case field.FTListInt64:
		return applyListRemoveNumber(s, fieldNum, index, 8, field.FTListInt64)
	case field.FTListUint64:
		return applyListRemoveNumber(s, fieldNum, index, 8, field.FTListUint64)
	case field.FTListFloat64:
		return applyListRemoveNumber(s, fieldNum, index, 8, field.FTListFloat64)
	case field.FTListBytes:
		return applyListRemoveBytes(s, fieldNum, index)
	case field.FTListStrings:
		return applyListRemoveString(s, fieldNum, index)
	case field.FTListStructs:
		return applyListRemoveStruct(s, fd, fieldNum, index)
	default:
		return fmt.Errorf("LIST_REMOVE not supported for field type: %v", fd.Type)
	}
}

func applyListRemoveBool(s *segment.Struct, fieldNum uint16, index int32) error {
	list := segment.GetListBools(s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	if len(existing) == 1 {
		segment.ClearField(s, fieldNum)
		return nil
	}
	// Allocate new slice to avoid mutating the underlying array
	newSlice := make([]bool, 0, len(existing)-1)
	newSlice = append(newSlice, existing[:index]...)
	newSlice = append(newSlice, existing[index+1:]...)
	newList := segment.NewBools(s, fieldNum)
	newList.SetAll(newSlice)
	return nil
}

func applyListRemoveNumber(s *segment.Struct, fieldNum uint16, index int32, sizeInBytes int, ft field.Type) error {
	// Get existing item data
	itemData, err := getNumberListItemData(s, fieldNum, sizeInBytes)
	if err != nil {
		return err
	}

	numItems := len(itemData) / sizeInBytes
	if int(index) >= numItems {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}

	if numItems == 1 {
		segment.ClearField(s, fieldNum)
		s.ClearListCache(fieldNum)
		return nil
	}

	// Remove the item at index
	removeOffset := int(index) * sizeInBytes
	newData := make([]byte, len(itemData)-sizeInBytes)
	copy(newData, itemData[:removeOffset])
	copy(newData[removeOffset:], itemData[removeOffset+sizeInBytes:])

	segment.SetListNumbersRaw(s, fieldNum, ft, newData)
	return nil
}

func applyListRemoveBytes(s *segment.Struct, fieldNum uint16, index int32) error {
	list := segment.GetListBytes(s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	if len(existing) == 1 {
		segment.ClearField(s, fieldNum)
		return nil
	}
	// Allocate new slice to avoid mutating the underlying array
	newSlice := make([][]byte, 0, len(existing)-1)
	newSlice = append(newSlice, existing[:index]...)
	newSlice = append(newSlice, existing[index+1:]...)
	newList := segment.NewBytes(s, fieldNum)
	newList.SetAll(newSlice)
	return nil
}

func applyListRemoveString(s *segment.Struct, fieldNum uint16, index int32) error {
	list := segment.GetListStrings(s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	if len(existing) == 1 {
		segment.ClearField(s, fieldNum)
		return nil
	}
	// Allocate new slice to avoid mutating the underlying array
	newSlice := make([]string, 0, len(existing)-1)
	newSlice = append(newSlice, existing[:index]...)
	newSlice = append(newSlice, existing[index+1:]...)
	newList := segment.NewStrings(s, fieldNum)
	newList.SetAll(newSlice)
	return nil
}

func applyListRemoveStruct(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32) error {
	list := segment.GetListStructs(s, fieldNum, fd.Mapping)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	if len(existing) == 1 {
		segment.ClearField(s, fieldNum)
		return nil
	}
	// Create new list without the item at index
	newList := segment.NewStructs(s, fieldNum, fd.Mapping)
	for i, item := range existing {
		if i == int(index) {
			continue
		}
		newList.Append(item)
	}
	return nil
}

func applyListStructPatch(s *segment.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte, depth int) error {
	if depth >= MaxPatchNestingDepth {
		return fmt.Errorf("patch nesting depth %d exceeds maximum of %d", depth, MaxPatchNestingDepth)
	}

	if fd == nil || fd.Type != field.FTListStructs {
		return fmt.Errorf("LIST_STRUCT_PATCH requires list of structs field")
	}

	// Get the list of structs
	list := segment.GetListStructs(s, fieldNum, fd.Mapping)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("list index %d out of bounds", index)
	}

	// Get the struct at the index
	itemStruct := list.Get(int(index))
	if itemStruct == nil {
		return fmt.Errorf("nil struct at index %d", index)
	}

	// Unmarshal the patch
	subPatch := msgs.NewPatch(nil)
	if err := subPatch.Unmarshal(data); err != nil {
		return fmt.Errorf("unmarshal sub-patch: %w", err)
	}

	// Apply each operation to the struct at the index
	opsLen := subPatch.OpsLen()
	for i := 0; i < opsLen; i++ {
		op := subPatch.OpsGet(i)
		if err := applyOpWithDepth(itemStruct, fd.Mapping, op, depth+1); err != nil {
			return err
		}
	}

	return nil
}
