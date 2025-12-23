package patch

import (
	"bytes"
	"fmt"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/structs"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

// Apply applies the patch to 'base', modifying it in place.
// Returns error if patch cannot be applied.
func Apply[T ClawStruct](base T, p msgs.Patch) error {
	s := base.XXXGetStruct()
	m := s.Map()

	for _, op := range p.Ops() {
		if err := applyOp(s, m, op); err != nil {
			return err
		}
	}

	return nil
}

// applyOp applies a single operation to the struct.
func applyOp(s *structs.Struct, m *mapping.Map, op msgs.Op) error {
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
		return applyStructPatch(s, fd, fieldNum, data)
	case msgs.ListReplace:
		return applyListReplace(s, fd, fieldNum, data)
	case msgs.ListSet:
		return applyListSet(s, fd, fieldNum, op.Index(), data)
	case msgs.ListInsert:
		return applyListInsert(s, fd, fieldNum, op.Index(), data)
	case msgs.ListRemove:
		return applyListRemove(s, fd, fieldNum, op.Index())
	case msgs.ListStructPatch:
		return applyListStructPatch(s, fd, fieldNum, op.Index(), data)
	default:
		// Unknown operation type - skip for forward compatibility
		return nil
	}
}

func applySet(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte) error {
	if fd == nil {
		// Unknown field - store as raw data for forward compatibility
		s.XXXSetUnknownField(fieldNum, data)
		return nil
	}

	switch fd.Type {
	case field.FTBool:
		if len(data) < 1 {
			return fmt.Errorf("bool data too short")
		}
		structs.MustSetBool(s, fieldNum, data[0] != 0)
	case field.FTInt8:
		if len(data) < 1 {
			return fmt.Errorf("int8 data too short")
		}
		structs.MustSetNumber(s, fieldNum, int8(data[0]))
	case field.FTUint8:
		if len(data) < 1 {
			return fmt.Errorf("uint8 data too short")
		}
		structs.MustSetNumber(s, fieldNum, data[0])
	case field.FTInt16:
		if len(data) < 2 {
			return fmt.Errorf("int16 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeInt16(data))
	case field.FTUint16:
		if len(data) < 2 {
			return fmt.Errorf("uint16 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeUint16(data))
	case field.FTInt32:
		if len(data) < 4 {
			return fmt.Errorf("int32 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeInt32(data))
	case field.FTUint32:
		if len(data) < 4 {
			return fmt.Errorf("uint32 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeUint32(data))
	case field.FTInt64:
		if len(data) < 8 {
			return fmt.Errorf("int64 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeInt64(data))
	case field.FTUint64:
		if len(data) < 8 {
			return fmt.Errorf("uint64 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeUint64(data))
	case field.FTFloat32:
		if len(data) < 4 {
			return fmt.Errorf("float32 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeFloat32(data))
	case field.FTFloat64:
		if len(data) < 8 {
			return fmt.Errorf("float64 data too short")
		}
		structs.MustSetNumber(s, fieldNum, decodeFloat64(data))
	case field.FTString:
		structs.MustSetBytes(s, fieldNum, data, true)
	case field.FTBytes:
		structs.MustSetBytes(s, fieldNum, data, false)
	case field.FTStruct:
		// For SET on struct, unmarshal the full struct
		subStruct := structs.New(0, fd.Mapping)
		if _, err := subStruct.Unmarshal(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("unmarshal nested struct: %w", err)
		}
		structs.MustSetStruct(s, fieldNum, subStruct)
	default:
		return fmt.Errorf("SET not supported for field type: %v", fd.Type)
	}

	return nil
}

func applyClear(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16) error {
	if fd == nil {
		// Unknown field - remove from forward compatibility storage
		s.XXXDeleteUnknownField(fieldNum)
		return nil
	}

	switch fd.Type {
	case field.FTBool:
		structs.MustSetBool(s, fieldNum, false)
	case field.FTInt8:
		structs.MustSetNumber(s, fieldNum, int8(0))
	case field.FTUint8:
		structs.MustSetNumber(s, fieldNum, uint8(0))
	case field.FTInt16:
		structs.MustSetNumber(s, fieldNum, int16(0))
	case field.FTUint16:
		structs.MustSetNumber(s, fieldNum, uint16(0))
	case field.FTInt32:
		structs.MustSetNumber(s, fieldNum, int32(0))
	case field.FTUint32:
		structs.MustSetNumber(s, fieldNum, uint32(0))
	case field.FTInt64:
		structs.MustSetNumber(s, fieldNum, int64(0))
	case field.FTUint64:
		structs.MustSetNumber(s, fieldNum, uint64(0))
	case field.FTFloat32:
		structs.MustSetNumber(s, fieldNum, float32(0))
	case field.FTFloat64:
		structs.MustSetNumber(s, fieldNum, float64(0))
	case field.FTString, field.FTBytes:
		structs.DeleteBytes(s, fieldNum)
	case field.FTStruct:
		structs.DeleteStruct(s, fieldNum)
	case field.FTListBools:
		structs.DeleteListBools(s, fieldNum)
	case field.FTListInt8:
		structs.DeleteListNumber[int8](s, fieldNum)
	case field.FTListInt16:
		structs.DeleteListNumber[int16](s, fieldNum)
	case field.FTListInt32:
		structs.DeleteListNumber[int32](s, fieldNum)
	case field.FTListInt64:
		structs.DeleteListNumber[int64](s, fieldNum)
	case field.FTListUint8:
		structs.DeleteListNumber[uint8](s, fieldNum)
	case field.FTListUint16:
		structs.DeleteListNumber[uint16](s, fieldNum)
	case field.FTListUint32:
		structs.DeleteListNumber[uint32](s, fieldNum)
	case field.FTListUint64:
		structs.DeleteListNumber[uint64](s, fieldNum)
	case field.FTListFloat32:
		structs.DeleteListNumber[float32](s, fieldNum)
	case field.FTListFloat64:
		structs.DeleteListNumber[float64](s, fieldNum)
	case field.FTListBytes, field.FTListStrings:
		structs.DeleteListBytes(s, fieldNum)
	case field.FTListStructs:
		structs.DeleteListStructs(s, fieldNum)
	default:
		return fmt.Errorf("CLEAR not supported for field type: %v", fd.Type)
	}

	return nil
}

func applyStructPatch(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte) error {
	if fd == nil || fd.Type != field.FTStruct {
		return fmt.Errorf("STRUCT_PATCH requires struct field")
	}

	// Get the existing nested struct
	subStruct := structs.MustGetStruct(s, fieldNum)
	if subStruct == nil {
		// Create new struct if doesn't exist
		subStruct = structs.New(0, fd.Mapping)
		structs.MustSetStruct(s, fieldNum, subStruct)
	}

	// Unmarshal the patch
	var subPatch msgs.Patch
	if err := subPatch.Unmarshal(data); err != nil {
		return fmt.Errorf("unmarshal sub-patch: %w", err)
	}

	// Apply each operation to the nested struct
	for _, op := range subPatch.Ops() {
		if err := applyOp(subStruct, fd.Mapping, op); err != nil {
			return err
		}
	}

	return nil
}

func applyListReplace(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte) error {
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
	case field.FTListInt8:
		return applyListReplaceNumbers[int8](s, fieldNum, data, 1)
	case field.FTListUint8:
		return applyListReplaceNumbers[uint8](s, fieldNum, data, 1)
	case field.FTListInt16:
		return applyListReplaceNumbers[int16](s, fieldNum, data, 2)
	case field.FTListUint16:
		return applyListReplaceNumbers[uint16](s, fieldNum, data, 2)
	case field.FTListInt32:
		return applyListReplaceNumbers[int32](s, fieldNum, data, 4)
	case field.FTListUint32:
		return applyListReplaceNumbers[uint32](s, fieldNum, data, 4)
	case field.FTListFloat32:
		return applyListReplaceNumbers[float32](s, fieldNum, data, 4)
	case field.FTListInt64:
		return applyListReplaceNumbers[int64](s, fieldNum, data, 8)
	case field.FTListUint64:
		return applyListReplaceNumbers[uint64](s, fieldNum, data, 8)
	case field.FTListFloat64:
		return applyListReplaceNumbers[float64](s, fieldNum, data, 8)
	case field.FTListBytes, field.FTListStrings:
		return applyListReplaceBytes(s, fieldNum, data)
	case field.FTListStructs:
		return applyListReplaceStructs(s, fd, fieldNum, data)
	default:
		return fmt.Errorf("LIST_REPLACE not supported for field type: %v", fd.Type)
	}
}

func applyListReplaceBools(s *structs.Struct, fieldNum uint16, data []byte) error {
	bools := structs.NewBools(fieldNum)
	vals := make([]bool, len(data))
	for i, b := range data {
		vals[i] = b != 0
	}
	bools.Append(vals...)
	structs.MustSetListBool(s, fieldNum, bools)
	return nil
}

func applyListReplaceNumbers[N structs.Number](s *structs.Struct, fieldNum uint16, data []byte, sizeInBytes int) error {
	if len(data)%sizeInBytes != 0 {
		return fmt.Errorf("invalid number list data length: %d not divisible by %d", len(data), sizeInBytes)
	}
	count := len(data) / sizeInBytes
	nums := structs.NewNumbers[N]()
	values := make([]N, count)
	for i := 0; i < count; i++ {
		offset := i * sizeInBytes
		switch sizeInBytes {
		case 1:
			values[i] = N(data[offset])
		case 2:
			values[i] = N(decodeUint16(data[offset:]))
		case 4:
			// Check if float by trying to detect if this is float32
			var zero N
			switch any(zero).(type) {
			case float32:
				values[i] = N(decodeFloat32(data[offset:]))
			default:
				values[i] = N(decodeUint32(data[offset:]))
			}
		case 8:
			var zero N
			switch any(zero).(type) {
			case float64:
				values[i] = N(decodeFloat64(data[offset:]))
			default:
				values[i] = N(decodeUint64(data[offset:]))
			}
		}
	}
	nums.Append(values...)
	structs.MustSetListNumber(s, fieldNum, nums)
	return nil
}

func applyListReplaceBytes(s *structs.Struct, fieldNum uint16, data []byte) error {
	b := structs.NewBytes()
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
	b.Append(items...)
	structs.MustSetListBytes(s, fieldNum, b)
	return nil
}

func applyListReplaceStructs(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, data []byte) error {
	list := structs.NewStructs(fd.Mapping)
	// Data format: [count:4][struct1...][struct2...]...
	if len(data) < 4 {
		return fmt.Errorf("struct list data too short")
	}
	count := int(decodeUint32(data[:4]))
	reader := bytes.NewReader(data[4:])
	items := make([]*structs.Struct, 0, count)
	for i := 0; i < count; i++ {
		item := structs.New(0, fd.Mapping)
		if _, err := item.Unmarshal(reader); err != nil {
			return fmt.Errorf("unmarshal struct at index %d: %w", i, err)
		}
		items = append(items, item)
	}
	if err := list.Append(items...); err != nil {
		return fmt.Errorf("append structs: %w", err)
	}
	structs.MustSetListStruct(s, fieldNum, list)
	return nil
}

func applyListSet(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	if fd == nil {
		return nil // Unknown field, skip
	}
	if index < 0 {
		return fmt.Errorf("LIST_SET invalid index: %d", index)
	}

	switch fd.Type {
	case field.FTListBools:
		list := structs.MustGetListBool(s, fieldNum)
		if list == nil || int(index) >= list.Len() {
			return fmt.Errorf("LIST_SET index %d out of bounds", index)
		}
		if len(data) < 1 {
			return fmt.Errorf("LIST_SET bool data too short")
		}
		list.Set(int(index), data[0] != 0)
	case field.FTListInt8:
		return applyListSetNumber[int8](s, fieldNum, index, data, 1)
	case field.FTListUint8:
		return applyListSetNumber[uint8](s, fieldNum, index, data, 1)
	case field.FTListInt16:
		return applyListSetNumber[int16](s, fieldNum, index, data, 2)
	case field.FTListUint16:
		return applyListSetNumber[uint16](s, fieldNum, index, data, 2)
	case field.FTListInt32:
		return applyListSetNumber[int32](s, fieldNum, index, data, 4)
	case field.FTListUint32:
		return applyListSetNumber[uint32](s, fieldNum, index, data, 4)
	case field.FTListFloat32:
		return applyListSetNumber[float32](s, fieldNum, index, data, 4)
	case field.FTListInt64:
		return applyListSetNumber[int64](s, fieldNum, index, data, 8)
	case field.FTListUint64:
		return applyListSetNumber[uint64](s, fieldNum, index, data, 8)
	case field.FTListFloat64:
		return applyListSetNumber[float64](s, fieldNum, index, data, 8)
	case field.FTListBytes, field.FTListStrings:
		list := structs.MustGetListBytes(s, fieldNum)
		if list == nil || int(index) >= list.Len() {
			return fmt.Errorf("LIST_SET index %d out of bounds", index)
		}
		list.Set(int(index), data)
	case field.FTListStructs:
		list := structs.MustGetListStruct(s, fieldNum)
		if list == nil || int(index) >= list.Len() {
			return fmt.Errorf("LIST_SET index %d out of bounds", index)
		}
		item := structs.New(0, fd.Mapping)
		if _, err := item.Unmarshal(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("unmarshal struct for LIST_SET: %w", err)
		}
		if err := list.Set(int(index), item); err != nil {
			return fmt.Errorf("LIST_SET struct: %w", err)
		}
	default:
		return fmt.Errorf("LIST_SET not supported for field type: %v", fd.Type)
	}
	return nil
}

func applyListSetNumber[N structs.Number](s *structs.Struct, fieldNum uint16, index int32, data []byte, sizeInBytes int) error {
	list := structs.MustGetListNumber[N](s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_SET index %d out of bounds", index)
	}
	if len(data) < sizeInBytes {
		return fmt.Errorf("LIST_SET number data too short")
	}
	var value N
	switch sizeInBytes {
	case 1:
		value = N(data[0])
	case 2:
		value = N(decodeUint16(data))
	case 4:
		var zero N
		switch any(zero).(type) {
		case float32:
			value = N(decodeFloat32(data))
		default:
			value = N(decodeUint32(data))
		}
	case 8:
		var zero N
		switch any(zero).(type) {
		case float64:
			value = N(decodeFloat64(data))
		default:
			value = N(decodeUint64(data))
		}
	}
	list.Set(int(index), value)
	return nil
}

func applyListInsert(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
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
		return applyListInsertNumber[int8](s, fieldNum, index, data, 1)
	case field.FTListUint8:
		return applyListInsertNumber[uint8](s, fieldNum, index, data, 1)
	case field.FTListInt16:
		return applyListInsertNumber[int16](s, fieldNum, index, data, 2)
	case field.FTListUint16:
		return applyListInsertNumber[uint16](s, fieldNum, index, data, 2)
	case field.FTListInt32:
		return applyListInsertNumber[int32](s, fieldNum, index, data, 4)
	case field.FTListUint32:
		return applyListInsertNumber[uint32](s, fieldNum, index, data, 4)
	case field.FTListFloat32:
		return applyListInsertNumber[float32](s, fieldNum, index, data, 4)
	case field.FTListInt64:
		return applyListInsertNumber[int64](s, fieldNum, index, data, 8)
	case field.FTListUint64:
		return applyListInsertNumber[uint64](s, fieldNum, index, data, 8)
	case field.FTListFloat64:
		return applyListInsertNumber[float64](s, fieldNum, index, data, 8)
	case field.FTListBytes, field.FTListStrings:
		return applyListInsertBytes(s, fieldNum, index, data)
	case field.FTListStructs:
		return applyListInsertStruct(s, fd, fieldNum, index, data)
	default:
		return fmt.Errorf("LIST_INSERT not supported for field type: %v", fd.Type)
	}
}

func applyListInsertBool(s *structs.Struct, fieldNum uint16, index int32, data []byte) error {
	list := structs.MustGetListBool(s, fieldNum)
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
	// Insert at index
	newSlice := make([]bool, 0, len(existing)+1)
	newSlice = append(newSlice, existing[:index]...)
	newSlice = append(newSlice, newVal)
	newSlice = append(newSlice, existing[index:]...)

	newList := structs.NewBools(fieldNum)
	newList.Append(newSlice...)
	structs.MustSetListBool(s, fieldNum, newList)
	return nil
}

func applyListInsertNumber[N structs.Number](s *structs.Struct, fieldNum uint16, index int32, data []byte, sizeInBytes int) error {
	list := structs.MustGetListNumber[N](s, fieldNum)
	var existing []N
	if list != nil {
		existing = list.Slice()
	}
	if int(index) > len(existing) {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, len(existing))
	}
	if len(data) < sizeInBytes {
		return fmt.Errorf("LIST_INSERT number data too short")
	}
	var newVal N
	switch sizeInBytes {
	case 1:
		newVal = N(data[0])
	case 2:
		newVal = N(decodeUint16(data))
	case 4:
		var zero N
		switch any(zero).(type) {
		case float32:
			newVal = N(decodeFloat32(data))
		default:
			newVal = N(decodeUint32(data))
		}
	case 8:
		var zero N
		switch any(zero).(type) {
		case float64:
			newVal = N(decodeFloat64(data))
		default:
			newVal = N(decodeUint64(data))
		}
	}
	// Insert at index
	newSlice := make([]N, 0, len(existing)+1)
	newSlice = append(newSlice, existing[:index]...)
	newSlice = append(newSlice, newVal)
	newSlice = append(newSlice, existing[index:]...)

	newList := structs.NewNumbers[N]()
	newList.Append(newSlice...)
	structs.MustSetListNumber(s, fieldNum, newList)
	return nil
}

func applyListInsertBytes(s *structs.Struct, fieldNum uint16, index int32, data []byte) error {
	list := structs.MustGetListBytes(s, fieldNum)
	var existing [][]byte
	if list != nil {
		existing = list.Slice()
	}
	if int(index) > len(existing) {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, len(existing))
	}
	// Insert at index
	newSlice := make([][]byte, 0, len(existing)+1)
	newSlice = append(newSlice, existing[:index]...)
	newSlice = append(newSlice, data)
	newSlice = append(newSlice, existing[index:]...)

	newList := structs.NewBytes()
	newList.Append(newSlice...)
	structs.MustSetListBytes(s, fieldNum, newList)
	return nil
}

func applyListInsertStruct(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	list := structs.MustGetListStruct(s, fieldNum)
	var existing []*structs.Struct
	if list != nil {
		existing = list.Slice()
	}
	if int(index) > len(existing) {
		return fmt.Errorf("LIST_INSERT index %d out of bounds (len=%d)", index, len(existing))
	}
	item := structs.New(0, fd.Mapping)
	if _, err := item.Unmarshal(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("unmarshal struct for LIST_INSERT: %w", err)
	}
	// Insert at index - need to create new slice since existing items have parents
	newList := structs.NewStructs(fd.Mapping)
	// Append items before index
	for i := 0; i < int(index); i++ {
		clone := cloneStruct(existing[i], fd.Mapping)
		if err := newList.Append(clone); err != nil {
			return fmt.Errorf("append clone: %w", err)
		}
	}
	// Append new item
	if err := newList.Append(item); err != nil {
		return fmt.Errorf("append new item: %w", err)
	}
	// Append items after index
	for i := int(index); i < len(existing); i++ {
		clone := cloneStruct(existing[i], fd.Mapping)
		if err := newList.Append(clone); err != nil {
			return fmt.Errorf("append clone: %w", err)
		}
	}
	structs.MustSetListStruct(s, fieldNum, newList)
	return nil
}

func applyListRemove(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32) error {
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
		return applyListRemoveNumber[int8](s, fieldNum, index)
	case field.FTListUint8:
		return applyListRemoveNumber[uint8](s, fieldNum, index)
	case field.FTListInt16:
		return applyListRemoveNumber[int16](s, fieldNum, index)
	case field.FTListUint16:
		return applyListRemoveNumber[uint16](s, fieldNum, index)
	case field.FTListInt32:
		return applyListRemoveNumber[int32](s, fieldNum, index)
	case field.FTListUint32:
		return applyListRemoveNumber[uint32](s, fieldNum, index)
	case field.FTListFloat32:
		return applyListRemoveNumber[float32](s, fieldNum, index)
	case field.FTListInt64:
		return applyListRemoveNumber[int64](s, fieldNum, index)
	case field.FTListUint64:
		return applyListRemoveNumber[uint64](s, fieldNum, index)
	case field.FTListFloat64:
		return applyListRemoveNumber[float64](s, fieldNum, index)
	case field.FTListBytes, field.FTListStrings:
		return applyListRemoveBytes(s, fieldNum, index)
	case field.FTListStructs:
		return applyListRemoveStruct(s, fd, fieldNum, index)
	default:
		return fmt.Errorf("LIST_REMOVE not supported for field type: %v", fd.Type)
	}
}

func applyListRemoveBool(s *structs.Struct, fieldNum uint16, index int32) error {
	list := structs.MustGetListBool(s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	newSlice := append(existing[:index], existing[index+1:]...)
	if len(newSlice) == 0 {
		structs.DeleteListBools(s, fieldNum)
		return nil
	}
	newList := structs.NewBools(fieldNum)
	newList.Append(newSlice...)
	structs.MustSetListBool(s, fieldNum, newList)
	return nil
}

func applyListRemoveNumber[N structs.Number](s *structs.Struct, fieldNum uint16, index int32) error {
	list := structs.MustGetListNumber[N](s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	newSlice := append(existing[:index], existing[index+1:]...)
	if len(newSlice) == 0 {
		structs.DeleteListNumber[N](s, fieldNum)
		return nil
	}
	newList := structs.NewNumbers[N]()
	newList.Append(newSlice...)
	structs.MustSetListNumber(s, fieldNum, newList)
	return nil
}

func applyListRemoveBytes(s *structs.Struct, fieldNum uint16, index int32) error {
	list := structs.MustGetListBytes(s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	newSlice := append(existing[:index], existing[index+1:]...)
	if len(newSlice) == 0 {
		structs.DeleteListBytes(s, fieldNum)
		return nil
	}
	newList := structs.NewBytes()
	newList.Append(newSlice...)
	structs.MustSetListBytes(s, fieldNum, newList)
	return nil
}

func applyListRemoveStruct(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32) error {
	list := structs.MustGetListStruct(s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("LIST_REMOVE index %d out of bounds", index)
	}
	existing := list.Slice()
	if len(existing) == 1 {
		structs.DeleteListStructs(s, fieldNum)
		return nil
	}
	// Create new list without the item at index
	newList := structs.NewStructs(fd.Mapping)
	for i, item := range existing {
		if i == int(index) {
			continue
		}
		clone := cloneStruct(item, fd.Mapping)
		if err := newList.Append(clone); err != nil {
			return fmt.Errorf("append clone: %w", err)
		}
	}
	structs.MustSetListStruct(s, fieldNum, newList)
	return nil
}

// cloneStruct creates a deep copy of a struct by marshaling and unmarshaling it.
func cloneStruct(src *structs.Struct, m *mapping.Map) *structs.Struct {
	buf := &bytes.Buffer{}
	if _, err := src.Marshal(buf); err != nil {
		panic(fmt.Sprintf("clone marshal failed: %v", err))
	}
	dst := structs.New(0, m)
	if _, err := dst.Unmarshal(bytes.NewReader(buf.Bytes())); err != nil {
		panic(fmt.Sprintf("clone unmarshal failed: %v", err))
	}
	return dst
}

func applyListStructPatch(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	if fd == nil || fd.Type != field.FTListStructs {
		return fmt.Errorf("LIST_STRUCT_PATCH requires list of structs field")
	}

	// Get the list of structs
	list := structs.MustGetListStruct(s, fieldNum)
	if list == nil || int(index) >= list.Len() {
		return fmt.Errorf("list index %d out of bounds", index)
	}

	// Get the struct at the index
	itemStruct := list.Get(int(index))
	if itemStruct == nil {
		return fmt.Errorf("nil struct at index %d", index)
	}

	// Unmarshal the patch
	var subPatch msgs.Patch
	if err := subPatch.Unmarshal(data); err != nil {
		return fmt.Errorf("unmarshal sub-patch: %w", err)
	}

	// Apply each operation to the struct at the index
	for _, op := range subPatch.Ops() {
		if err := applyOp(itemStruct, fd.Mapping, op); err != nil {
			return err
		}
	}

	return nil
}
