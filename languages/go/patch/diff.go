// Package patch provides diffing and patching functionality for Claw structs.
package patch

import (
	"bytes"
	"fmt"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/structs"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

// PatchVersion is the current patch format version.
const PatchVersion = 1

// ClawStruct is the interface that all generated claw structs implement.
type ClawStruct interface {
	XXXGetStruct() *structs.Struct
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// NoPatcher is an optional interface implemented by structs with the NoPatch() option.
type NoPatcher interface {
	XXXHasNoPatch() bool
}

// Diff computes the patch to transform 'from' into 'to'.
// Both must be the same struct type.
// Returns error if the struct has NoPatch() option or if types don't match.
func Diff[T ClawStruct](from, to T) (msgs.Patch, error) {
	// Check if type has NoPatch option
	if noPatcher, ok := any(from).(NoPatcher); ok && noPatcher.XXXHasNoPatch() {
		return msgs.Patch{}, fmt.Errorf("struct type has NoPatch option and cannot be diffed")
	}

	fromS := from.XXXGetStruct()
	toS := to.XXXGetStruct()

	fromMap := fromS.Map()
	toMap := toS.Map()

	if fromMap.Name != toMap.Name || fromMap.Path != toMap.Path {
		return msgs.Patch{}, fmt.Errorf("struct types don't match: %s vs %s", fromMap.Name, toMap.Name)
	}

	patch := msgs.NewPatch()
	patch.SetVersion(PatchVersion)

	if err := diffStruct(fromS, toS, &patch); err != nil {
		return msgs.Patch{}, err
	}

	return patch, nil
}

// diffStruct compares two structs and appends operations to the patch.
func diffStruct(from, to *structs.Struct, patch *msgs.Patch) error {
	m := from.Map()

	// Diff known fields
	for _, fd := range m.Fields {
		if fd == nil {
			continue
		}
		fieldNum := fd.FieldNum

		op, err := diffField(from, to, fd)
		if err != nil {
			return fmt.Errorf("field %d (%s): %w", fieldNum, fd.Name, err)
		}
		if op != nil {
			patch.AppendOps(*op)
		}
	}

	// Diff unknown fields for forward compatibility
	if err := diffUnknownFields(from, to, patch); err != nil {
		return err
	}

	return nil
}

// diffUnknownFields handles fields that exist in the wire format but are not in the mapping.
// This enables forward compatibility - changes to unknown fields are preserved in the patch.
func diffUnknownFields(from, to *structs.Struct, patch *msgs.Patch) error {
	fromUnknown := from.XXXUnknownFields()
	toUnknown := to.XXXUnknownFields()

	// Build a map of unknown fields in 'from' by field number
	fromMap := make(map[uint16][]byte)
	for _, f := range fromUnknown {
		fromMap[f.FieldNum] = f.Data
	}

	// Build a map of unknown fields in 'to' by field number
	toMap := make(map[uint16][]byte)
	for _, f := range toUnknown {
		toMap[f.FieldNum] = f.Data
	}

	// Find fields that exist in 'from' but not in 'to' (CLEAR operations)
	for fieldNum := range fromMap {
		if _, exists := toMap[fieldNum]; !exists {
			op := msgs.NewOp()
			op.SetFieldNum(fieldNum)
			op.SetType(msgs.Clear)
			op.SetIndex(-1)
			patch.AppendOps(op)
		}
	}

	// Find fields that exist in 'to' but not in 'from', or that differ (SET operations)
	for fieldNum, toData := range toMap {
		fromData, existsInFrom := fromMap[fieldNum]
		if !existsInFrom || !bytes.Equal(fromData, toData) {
			op := msgs.NewOp()
			op.SetFieldNum(fieldNum)
			op.SetType(msgs.Set)
			op.SetIndex(-1)
			op.SetData(toData)
			patch.AppendOps(op)
		}
	}

	return nil
}

// diffField compares a single field and returns an Op if different, nil if same.
func diffField(from, to *structs.Struct, fd *mapping.FieldDescr) (*msgs.Op, error) {
	fieldNum := fd.FieldNum

	switch fd.Type {
	case field.FTBool:
		return diffBool(from, to, fieldNum)
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64,
		field.FTUint8, field.FTUint16, field.FTUint32, field.FTUint64,
		field.FTFloat32, field.FTFloat64:
		return diffNumber(from, to, fieldNum, fd)
	case field.FTString, field.FTBytes:
		return diffBytes(from, to, fieldNum, fd.Type == field.FTString)
	case field.FTStruct:
		return diffNestedStruct(from, to, fieldNum, fd)
	case field.FTListBools:
		return diffListBool(from, to, fieldNum)
	case field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
		field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
		field.FTListFloat32, field.FTListFloat64:
		return diffListNumber(from, to, fieldNum, fd)
	case field.FTListBytes, field.FTListStrings:
		return diffListBytes(from, to, fieldNum)
	case field.FTListStructs:
		return diffListStructs(from, to, fieldNum, fd)
	default:
		return nil, fmt.Errorf("unsupported field type: %v", fd.Type)
	}
}

func diffBool(from, to *structs.Struct, fieldNum uint16) (*msgs.Op, error) {
	fromVal := structs.MustGetBool(from, fieldNum)
	toVal := structs.MustGetBool(to, fieldNum)

	if fromVal == toVal {
		return nil, nil
	}

	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.Set)
	op.SetIndex(-1)
	// Encode bool as single byte
	if toVal {
		op.SetData([]byte{1})
	} else {
		op.SetData([]byte{0})
	}
	return &op, nil
}

func diffNumber(from, to *structs.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	var fromBytes, toBytes []byte
	var equal bool

	switch fd.Type {
	case field.FTInt8:
		fromVal := structs.MustGetNumber[int8](from, fieldNum)
		toVal := structs.MustGetNumber[int8](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = []byte{byte(toVal)}
		}
	case field.FTUint8:
		fromVal := structs.MustGetNumber[uint8](from, fieldNum)
		toVal := structs.MustGetNumber[uint8](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = []byte{toVal}
		}
	case field.FTInt16:
		fromVal := structs.MustGetNumber[int16](from, fieldNum)
		toVal := structs.MustGetNumber[int16](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeInt16(toVal)
		}
	case field.FTUint16:
		fromVal := structs.MustGetNumber[uint16](from, fieldNum)
		toVal := structs.MustGetNumber[uint16](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeUint16(toVal)
		}
	case field.FTInt32:
		fromVal := structs.MustGetNumber[int32](from, fieldNum)
		toVal := structs.MustGetNumber[int32](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeInt32(toVal)
		}
	case field.FTUint32:
		fromVal := structs.MustGetNumber[uint32](from, fieldNum)
		toVal := structs.MustGetNumber[uint32](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeUint32(toVal)
		}
	case field.FTInt64:
		fromVal := structs.MustGetNumber[int64](from, fieldNum)
		toVal := structs.MustGetNumber[int64](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeInt64(toVal)
		}
	case field.FTUint64:
		fromVal := structs.MustGetNumber[uint64](from, fieldNum)
		toVal := structs.MustGetNumber[uint64](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeUint64(toVal)
		}
	case field.FTFloat32:
		fromVal := structs.MustGetNumber[float32](from, fieldNum)
		toVal := structs.MustGetNumber[float32](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeFloat32(toVal)
		}
	case field.FTFloat64:
		fromVal := structs.MustGetNumber[float64](from, fieldNum)
		toVal := structs.MustGetNumber[float64](to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeFloat64(toVal)
		}
	default:
		return nil, fmt.Errorf("unexpected number type: %v", fd.Type)
	}

	if equal {
		return nil, nil
	}

	_ = fromBytes // unused for now, just comparing values

	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.Set)
	op.SetIndex(-1)
	op.SetData(toBytes)
	return &op, nil
}

func diffBytes(from, to *structs.Struct, fieldNum uint16, isString bool) (*msgs.Op, error) {
	fromPtr := structs.MustGetBytes(from, fieldNum)
	toPtr := structs.MustGetBytes(to, fieldNum)

	var fromVal, toVal []byte
	if fromPtr != nil {
		fromVal = *fromPtr
	}
	if toPtr != nil {
		toVal = *toPtr
	}

	if bytes.Equal(fromVal, toVal) {
		return nil, nil
	}

	// If toVal is nil/empty and fromVal was set, use CLEAR
	if len(toVal) == 0 && len(fromVal) > 0 {
		op := msgs.NewOp()
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.Clear)
		op.SetIndex(-1)
		return &op, nil
	}

	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.Set)
	op.SetIndex(-1)
	op.SetData(toVal)
	return &op, nil
}

func diffNestedStruct(from, to *structs.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	fromSub := structs.MustGetStruct(from, fieldNum)
	toSub := structs.MustGetStruct(to, fieldNum)

	// Handle nil transitions
	if fromSub == nil && toSub == nil {
		return nil, nil
	}
	if fromSub != nil && toSub == nil {
		// Nested struct was cleared
		op := msgs.NewOp()
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.Clear)
		op.SetIndex(-1)
		return &op, nil
	}
	if fromSub == nil && toSub != nil {
		// Nested struct was added - encode full struct
		buf := &bytes.Buffer{}
		if _, err := toSub.Marshal(buf); err != nil {
			return nil, fmt.Errorf("marshal nested struct: %w", err)
		}
		op := msgs.NewOp()
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.Set)
		op.SetIndex(-1)
		op.SetData(buf.Bytes())
		return &op, nil
	}

	// Both exist - compute recursive patch
	subPatch := msgs.NewPatch()
	subPatch.SetVersion(PatchVersion)
	if err := diffStruct(fromSub, toSub, &subPatch); err != nil {
		return nil, fmt.Errorf("diff nested struct: %w", err)
	}

	// If no changes, return nil
	if len(subPatch.Ops()) == 0 {
		return nil, nil
	}

	// Serialize the sub-patch
	patchBytes, err := subPatch.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal sub-patch: %w", err)
	}

	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.StructPatch)
	op.SetIndex(-1)
	op.SetData(patchBytes)
	return &op, nil
}

func diffListBool(from, to *structs.Struct, fieldNum uint16) (*msgs.Op, error) {
	fromList := structs.MustGetListBool(from, fieldNum)
	toList := structs.MustGetListBool(to, fieldNum)

	var fromLen, toLen int
	if fromList != nil {
		fromLen = fromList.Len()
	}
	if toList != nil {
		toLen = toList.Len()
	}

	// If both empty or both nil, no change
	if fromLen == 0 && toLen == 0 {
		return nil, nil
	}

	// For simplicity, use LIST_REPLACE for boolean lists
	// (index-based ops on bool lists are less useful)
	return createListReplaceBools(fieldNum, toList)
}

func diffListNumber(from, to *structs.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	// For now, use LIST_REPLACE for number lists
	// Full index-based diffing can be added later
	return diffListNumberReplace(from, to, fieldNum, fd)
}

func diffListBytes(from, to *structs.Struct, fieldNum uint16) (*msgs.Op, error) {
	fromList := structs.MustGetListBytes(from, fieldNum)
	toList := structs.MustGetListBytes(to, fieldNum)

	var fromLen, toLen int
	if fromList != nil {
		fromLen = fromList.Len()
	}
	if toList != nil {
		toLen = toList.Len()
	}

	if fromLen == 0 && toLen == 0 {
		return nil, nil
	}

	// For now, use LIST_REPLACE
	return createListReplaceBytes(fieldNum, toList)
}

func diffListStructs(from, to *structs.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	fromList := structs.MustGetListStruct(from, fieldNum)
	toList := structs.MustGetListStruct(to, fieldNum)

	var fromLen, toLen int
	if fromList != nil {
		fromLen = fromList.Len()
	}
	if toList != nil {
		toLen = toList.Len()
	}

	if fromLen == 0 && toLen == 0 {
		return nil, nil
	}

	// For now, use LIST_REPLACE for struct lists
	// Full index-based diffing with LIST_STRUCT_PATCH can be added later
	return createListReplaceStructs(fieldNum, toList)
}

// Helper functions for list replace operations
func createListReplaceBools(fieldNum uint16, toList *structs.Bools) (*msgs.Op, error) {
	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(-1)

	if toList == nil || toList.Len() == 0 {
		op.SetData(nil)
		return &op, nil
	}

	// Encode as simple byte slice (1 byte per bool)
	data := make([]byte, toList.Len())
	for i := 0; i < toList.Len(); i++ {
		if toList.Get(i) {
			data[i] = 1
		}
	}
	op.SetData(data)
	return &op, nil
}

func diffListNumberReplace(from, to *structs.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(-1)

	// Encode based on the specific number type
	switch fd.Type {
	case field.FTListInt8:
		data, changed := encodeListNumber[int8](from, to, fieldNum, 1)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListUint8:
		data, changed := encodeListNumber[uint8](from, to, fieldNum, 1)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListInt16:
		data, changed := encodeListNumber[int16](from, to, fieldNum, 2)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListUint16:
		data, changed := encodeListNumber[uint16](from, to, fieldNum, 2)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListInt32:
		data, changed := encodeListNumber[int32](from, to, fieldNum, 4)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListUint32:
		data, changed := encodeListNumber[uint32](from, to, fieldNum, 4)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListFloat32:
		data, changed := encodeListNumberFloat32(from, to, fieldNum)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListInt64:
		data, changed := encodeListNumber[int64](from, to, fieldNum, 8)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListUint64:
		data, changed := encodeListNumber[uint64](from, to, fieldNum, 8)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	case field.FTListFloat64:
		data, changed := encodeListNumberFloat64(from, to, fieldNum)
		if !changed {
			return nil, nil
		}
		op.SetData(data)
	default:
		return nil, fmt.Errorf("unexpected number list type: %v", fd.Type)
	}

	return &op, nil
}

func encodeListNumber[N structs.Number](from, to *structs.Struct, fieldNum uint16, sizeInBytes int) ([]byte, bool) {
	fromList := structs.MustGetListNumber[N](from, fieldNum)
	toList := structs.MustGetListNumber[N](to, fieldNum)

	var fromSlice, toSlice []N
	if fromList != nil {
		fromSlice = fromList.Slice()
	}
	if toList != nil {
		toSlice = toList.Slice()
	}

	// Check if lists are equal
	if len(fromSlice) == len(toSlice) {
		equal := true
		for i := range fromSlice {
			if fromSlice[i] != toSlice[i] {
				equal = false
				break
			}
		}
		if equal {
			return nil, false
		}
	}

	// Encode the target list
	data := make([]byte, len(toSlice)*sizeInBytes)
	for i, v := range toSlice {
		offset := i * sizeInBytes
		switch sizeInBytes {
		case 1:
			data[offset] = byte(v)
		case 2:
			copy(data[offset:], encodeUint16(uint16(v)))
		case 4:
			copy(data[offset:], encodeUint32(uint32(v)))
		case 8:
			copy(data[offset:], encodeUint64(uint64(v)))
		}
	}
	return data, true
}

func encodeListNumberFloat32(from, to *structs.Struct, fieldNum uint16) ([]byte, bool) {
	fromList := structs.MustGetListNumber[float32](from, fieldNum)
	toList := structs.MustGetListNumber[float32](to, fieldNum)

	var fromSlice, toSlice []float32
	if fromList != nil {
		fromSlice = fromList.Slice()
	}
	if toList != nil {
		toSlice = toList.Slice()
	}

	// Check if lists are equal
	if len(fromSlice) == len(toSlice) {
		equal := true
		for i := range fromSlice {
			if fromSlice[i] != toSlice[i] {
				equal = false
				break
			}
		}
		if equal {
			return nil, false
		}
	}

	// Encode the target list
	data := make([]byte, len(toSlice)*4)
	for i, v := range toSlice {
		copy(data[i*4:], encodeFloat32(v))
	}
	return data, true
}

func encodeListNumberFloat64(from, to *structs.Struct, fieldNum uint16) ([]byte, bool) {
	fromList := structs.MustGetListNumber[float64](from, fieldNum)
	toList := structs.MustGetListNumber[float64](to, fieldNum)

	var fromSlice, toSlice []float64
	if fromList != nil {
		fromSlice = fromList.Slice()
	}
	if toList != nil {
		toSlice = toList.Slice()
	}

	// Check if lists are equal
	if len(fromSlice) == len(toSlice) {
		equal := true
		for i := range fromSlice {
			if fromSlice[i] != toSlice[i] {
				equal = false
				break
			}
		}
		if equal {
			return nil, false
		}
	}

	// Encode the target list
	data := make([]byte, len(toSlice)*8)
	for i, v := range toSlice {
		copy(data[i*8:], encodeFloat64(v))
	}
	return data, true
}

func createListReplaceBytes(fieldNum uint16, toList *structs.Bytes) (*msgs.Op, error) {
	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(-1)

	if toList == nil || toList.Len() == 0 {
		op.SetData(nil)
		return &op, nil
	}

	// Encode format: [count:4][len1:4][data1...][len2:4][data2...]...
	// Calculate total size
	totalSize := 4 // count
	for i := 0; i < toList.Len(); i++ {
		totalSize += 4 + len(toList.Get(i)) // length + data
	}

	data := make([]byte, totalSize)
	offset := 0

	// Write count
	copy(data[offset:], encodeUint32(uint32(toList.Len())))
	offset += 4

	// Write each item
	for i := 0; i < toList.Len(); i++ {
		item := toList.Get(i)
		copy(data[offset:], encodeUint32(uint32(len(item))))
		offset += 4
		copy(data[offset:], item)
		offset += len(item)
	}

	op.SetData(data)
	return &op, nil
}

func createListReplaceStructs(fieldNum uint16, toList *structs.Structs) (*msgs.Op, error) {
	op := msgs.NewOp()
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(-1)

	if toList == nil || toList.Len() == 0 {
		op.SetData(nil)
		return &op, nil
	}

	// Encode format: [count:4][struct1...][struct2...]...
	buf := &bytes.Buffer{}

	// Write count
	buf.Write(encodeUint32(uint32(toList.Len())))

	// Write each struct
	for i := 0; i < toList.Len(); i++ {
		item := toList.Get(i)
		if _, err := item.Marshal(buf); err != nil {
			return nil, fmt.Errorf("marshal struct at index %d: %w", i, err)
		}
	}

	op.SetData(buf.Bytes())
	return &op, nil
}

// IsEmpty returns true if the patch has no operations.
func IsEmpty(p msgs.Patch) bool {
	return len(p.Ops()) == 0
}
