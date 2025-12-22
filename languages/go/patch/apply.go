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

	// For now, implement basic list replacement
	// TODO: Full implementation for all list types
	switch fd.Type {
	case field.FTListBools:
		// Decode bools from data (1 byte per bool)
		bools := structs.NewBools(uint16(len(data)))
		for i, b := range data {
			bools.Set(i, b != 0)
		}
		structs.MustSetListBool(s, fieldNum, bools)
	case field.FTListStructs:
		// Clear existing and data is empty means empty list
		if len(data) == 0 {
			structs.DeleteListStructs(s, fieldNum)
		}
		// TODO: Decode full list of structs
	default:
		// For other list types, clear and leave empty for now
		// Full implementation would decode the list data
		return nil
	}

	return nil
}

func applyListSet(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	// TODO: Implement list set at index
	return nil
}

func applyListInsert(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32, data []byte) error {
	// TODO: Implement list insert at index
	return nil
}

func applyListRemove(s *structs.Struct, fd *mapping.FieldDescr, fieldNum uint16, index int32) error {
	// TODO: Implement list remove at index
	return nil
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
