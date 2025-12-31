// Package patch provides diffing and patching functionality for Claw structs.
package patch

import (
	"bytes"
	"fmt"
	"math"

	"github.com/gostdlib/base/context"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/segment"
	"github.com/bearlytools/claw/languages/go/patch/msgs"
)

// PatchVersion is the current patch format version.
const PatchVersion = 1

// NoListIndex indicates the operation is not list-indexed (for scalar field ops).
const NoListIndex int32 = -1

// ClawStruct is the interface that all generated claw structs implement.
type ClawStruct interface {
	XXXGetStruct() *segment.Struct
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
func Diff[T ClawStruct](ctx context.Context, from, to T) (msgs.Patch, error) {
	// Check if type has NoPatch option
	if noPatcher, ok := any(from).(NoPatcher); ok && noPatcher.XXXHasNoPatch() {
		return msgs.Patch{}, fmt.Errorf("struct type has NoPatch option and cannot be diffed")
	}

	fromS := from.XXXGetStruct()
	toS := to.XXXGetStruct()

	fromMap := fromS.Mapping()
	toMap := toS.Mapping()

	if fromMap.Name != toMap.Name || fromMap.Path != toMap.Path {
		return msgs.Patch{}, fmt.Errorf("struct types don't match: %s vs %s", fromMap.Name, toMap.Name)
	}

	patch := msgs.NewPatch(nil)
	patch.SetVersion(PatchVersion)

	if err := diffStructToPatch(ctx, fromS, toS, &patch); err != nil {
		return msgs.Patch{}, err
	}

	return patch, nil
}

// diffStructToPatch compares two structs and appends operations to the patch.
func diffStructToPatch(ctx context.Context, from, to *segment.Struct, patch *msgs.Patch) error {
	m := from.Mapping()

	// Diff known fields
	for _, fd := range m.Fields {
		if fd == nil {
			continue
		}
		fieldNum := fd.FieldNum

		ops, err := diffField(ctx, from, to, fd)
		if err != nil {
			return fmt.Errorf("field %d (%s): %w", fieldNum, fd.Name, err)
		}
		for _, op := range ops {
			patch.OpsAppend(ctx, op)
		}
	}

	// Note: Unknown fields not supported in segment currently
	// For forward compatibility, this would need to be added to segment.Struct

	return nil
}

// diffField compares a single field and returns Op(s) if different, nil if same.
// For scalar fields, returns at most one op. For list fields, may return multiple ops.
func diffField(ctx context.Context, from, to *segment.Struct, fd *mapping.FieldDescr) ([]msgs.Op, error) {
	fieldNum := fd.FieldNum

	switch fd.Type {
	case field.FTBool:
		op, err := diffBool(from, to, fieldNum)
		if err != nil || op == nil {
			return nil, err
		}
		return []msgs.Op{*op}, nil
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTInt64,
		field.FTUint8, field.FTUint16, field.FTUint32, field.FTUint64,
		field.FTFloat32, field.FTFloat64:
		op, err := diffNumber(from, to, fieldNum, fd)
		if err != nil || op == nil {
			return nil, err
		}
		return []msgs.Op{*op}, nil
	case field.FTString, field.FTBytes:
		op, err := diffBytes(from, to, fieldNum)
		if err != nil || op == nil {
			return nil, err
		}
		return []msgs.Op{*op}, nil
	case field.FTStruct:
		op, err := diffNestedStruct(ctx, from, to, fieldNum, fd)
		if err != nil || op == nil {
			return nil, err
		}
		return []msgs.Op{*op}, nil
	case field.FTListBools:
		return diffListBool(from, to, fieldNum)
	case field.FTListInt8, field.FTListInt16, field.FTListInt32, field.FTListInt64,
		field.FTListUint8, field.FTListUint16, field.FTListUint32, field.FTListUint64,
		field.FTListFloat32, field.FTListFloat64:
		return diffListNumber(from, to, fieldNum, fd)
	case field.FTListBytes, field.FTListStrings:
		return diffListBytes(from, to, fieldNum, fd.Type)
	case field.FTListStructs:
		return diffListStructs(ctx, from, to, fieldNum, fd)
	default:
		return nil, fmt.Errorf("unsupported field type: %v", fd.Type)
	}
}

func diffBool(from, to *segment.Struct, fieldNum uint16) (*msgs.Op, error) {
	fromVal := segment.GetBool(from, fieldNum)
	toVal := segment.GetBool(to, fieldNum)

	if fromVal == toVal {
		return nil, nil
	}

	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.Set)
	op.SetIndex(NoListIndex)
	// Encode bool as single byte
	if toVal {
		op.SetData([]byte{1})
	} else {
		op.SetData([]byte{0})
	}
	return &op, nil
}

func diffNumber(from, to *segment.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	var toBytes []byte
	var equal bool

	switch fd.Type {
	case field.FTInt8:
		fromVal := segment.GetInt8(from, fieldNum)
		toVal := segment.GetInt8(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = []byte{byte(toVal)}
		}
	case field.FTUint8:
		fromVal := segment.GetUint8(from, fieldNum)
		toVal := segment.GetUint8(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = []byte{toVal}
		}
	case field.FTInt16:
		fromVal := segment.GetInt16(from, fieldNum)
		toVal := segment.GetInt16(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeInt16(toVal)
		}
	case field.FTUint16:
		fromVal := segment.GetUint16(from, fieldNum)
		toVal := segment.GetUint16(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeUint16(toVal)
		}
	case field.FTInt32:
		fromVal := segment.GetInt32(from, fieldNum)
		toVal := segment.GetInt32(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeInt32(toVal)
		}
	case field.FTUint32:
		fromVal := segment.GetUint32(from, fieldNum)
		toVal := segment.GetUint32(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeUint32(toVal)
		}
	case field.FTInt64:
		fromVal := segment.GetInt64(from, fieldNum)
		toVal := segment.GetInt64(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeInt64(toVal)
		}
	case field.FTUint64:
		fromVal := segment.GetUint64(from, fieldNum)
		toVal := segment.GetUint64(to, fieldNum)
		equal = fromVal == toVal
		if !equal {
			toBytes = encodeUint64(toVal)
		}
	case field.FTFloat32:
		fromVal := segment.GetFloat32(from, fieldNum)
		toVal := segment.GetFloat32(to, fieldNum)
		// Handle NaN: both NaN is considered equal, otherwise use ==
		fromNaN := math.IsNaN(float64(fromVal))
		toNaN := math.IsNaN(float64(toVal))
		equal = (fromNaN && toNaN) || (!fromNaN && !toNaN && fromVal == toVal)
		if !equal {
			toBytes = encodeFloat32(toVal)
		}
	case field.FTFloat64:
		fromVal := segment.GetFloat64(from, fieldNum)
		toVal := segment.GetFloat64(to, fieldNum)
		// Handle NaN: both NaN is considered equal, otherwise use ==
		fromNaN := math.IsNaN(fromVal)
		toNaN := math.IsNaN(toVal)
		equal = (fromNaN && toNaN) || (!fromNaN && !toNaN && fromVal == toVal)
		if !equal {
			toBytes = encodeFloat64(toVal)
		}
	default:
		return nil, fmt.Errorf("unexpected number type: %v", fd.Type)
	}

	if equal {
		return nil, nil
	}

	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.Set)
	op.SetIndex(NoListIndex)
	op.SetData(toBytes)
	return &op, nil
}

func diffBytes(from, to *segment.Struct, fieldNum uint16) (*msgs.Op, error) {
	fromVal := segment.GetBytes(from, fieldNum)
	toVal := segment.GetBytes(to, fieldNum)

	if bytes.Equal(fromVal, toVal) {
		return nil, nil
	}

	// If toVal is nil/empty and fromVal was set, use CLEAR
	if len(toVal) == 0 && len(fromVal) > 0 {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.Clear)
		op.SetIndex(NoListIndex)
		return &op, nil
	}

	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.Set)
	op.SetIndex(NoListIndex)
	op.SetData(toVal)
	return &op, nil
}

func diffNestedStruct(ctx context.Context, from, to *segment.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	fromSub := segment.GetNestedStruct(from, fieldNum, fd.Mapping)
	toSub := segment.GetNestedStruct(to, fieldNum, fd.Mapping)

	// Handle nil transitions
	if fromSub == nil && toSub == nil {
		return nil, nil
	}
	if fromSub != nil && toSub == nil {
		// Nested struct was cleared
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.Clear)
		op.SetIndex(NoListIndex)
		return &op, nil
	}
	if fromSub == nil && toSub != nil {
		// Nested struct was added - encode full struct
		buf, err := toSub.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal nested struct: %w", err)
		}
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.Set)
		op.SetIndex(NoListIndex)
		op.SetData(buf)
		return &op, nil
	}

	// Both exist - compute recursive patch
	subPatch := msgs.NewPatch(nil)
	subPatch.SetVersion(PatchVersion)
	if err := diffStructToPatch(ctx, fromSub, toSub, &subPatch); err != nil {
		return nil, fmt.Errorf("diff nested struct: %w", err)
	}

	// If no changes, return nil
	if subPatch.OpsLen(ctx) == 0 {
		return nil, nil
	}

	// Serialize the sub-patch
	patchBytes, err := subPatch.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal sub-patch: %w", err)
	}

	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.StructPatch)
	op.SetIndex(NoListIndex)
	op.SetData(patchBytes)
	return &op, nil
}

func diffListBool(from, to *segment.Struct, fieldNum uint16) ([]msgs.Op, error) {
	fromList := segment.GetListBools(from, fieldNum)
	toList := segment.GetListBools(to, fieldNum)

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

	// Generate index-based operations
	var ops []msgs.Op

	// For indices that exist in both lists, generate SET if different
	minLen := fromLen
	if toLen < minLen {
		minLen = toLen
	}
	for i := 0; i < minLen; i++ {
		if fromList.Get(i) != toList.Get(i) {
			op := msgs.NewOp(nil)
			op.SetFieldNum(fieldNum)
			op.SetType(msgs.ListSet)
			op.SetIndex(int32(i))
			if toList.Get(i) {
				op.SetData([]byte{1})
			} else {
				op.SetData([]byte{0})
			}
			ops = append(ops, op)
		}
	}

	// For new indices in 'to', generate INSERT
	for i := fromLen; i < toLen; i++ {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListInsert)
		op.SetIndex(int32(i))
		if toList.Get(i) {
			op.SetData([]byte{1})
		} else {
			op.SetData([]byte{0})
		}
		ops = append(ops, op)
	}

	// For removed indices, generate REMOVE (in reverse order)
	for i := fromLen - 1; i >= toLen; i-- {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListRemove)
		op.SetIndex(int32(i))
		ops = append(ops, op)
	}

	// Use LIST_REPLACE when individual ops exceed half the combined list sizes.
	// This heuristic balances patch size vs granularity of changes.
	if len(ops) > 0 && len(ops) > (fromLen+toLen)/2 {
		replaceOp, err := createListReplaceBools(fieldNum, toList)
		if err != nil {
			return nil, err
		}
		if replaceOp != nil {
			return []msgs.Op{*replaceOp}, nil
		}
		return nil, nil
	}

	return ops, nil
}

func diffListNumber(from, to *segment.Struct, fieldNum uint16, fd *mapping.FieldDescr) ([]msgs.Op, error) {
	switch fd.Type {
	case field.FTListInt8:
		return diffListNumberTyped[int8](from, to, fieldNum, fd, 1, encodeInt8)
	case field.FTListUint8:
		return diffListNumberTyped[uint8](from, to, fieldNum, fd, 1, encodeUint8)
	case field.FTListInt16:
		return diffListNumberTyped[int16](from, to, fieldNum, fd, 2, encodeInt16)
	case field.FTListUint16:
		return diffListNumberTyped[uint16](from, to, fieldNum, fd, 2, encodeUint16)
	case field.FTListInt32:
		return diffListNumberTyped[int32](from, to, fieldNum, fd, 4, encodeInt32)
	case field.FTListUint32:
		return diffListNumberTyped[uint32](from, to, fieldNum, fd, 4, encodeUint32)
	case field.FTListFloat32:
		return diffListNumberFloat[float32](from, to, fieldNum, fd, encodeFloat32)
	case field.FTListInt64:
		return diffListNumberTyped[int64](from, to, fieldNum, fd, 8, encodeInt64)
	case field.FTListUint64:
		return diffListNumberTyped[uint64](from, to, fieldNum, fd, 8, encodeUint64)
	case field.FTListFloat64:
		return diffListNumberFloat[float64](from, to, fieldNum, fd, encodeFloat64)
	default:
		return nil, fmt.Errorf("unexpected number list type: %v", fd.Type)
	}
}

func diffListNumberTyped[N segment.Number](from, to *segment.Struct, fieldNum uint16, fd *mapping.FieldDescr, sizeInBytes int, encode func(N) []byte) ([]msgs.Op, error) {
	fromList := segment.GetListNumbers[N](from, fieldNum)
	toList := segment.GetListNumbers[N](to, fieldNum)

	var fromSlice, toSlice []N
	if fromList != nil {
		fromSlice = fromList.Slice()
	}
	if toList != nil {
		toSlice = toList.Slice()
	}

	fromLen := len(fromSlice)
	toLen := len(toSlice)

	if fromLen == 0 && toLen == 0 {
		return nil, nil
	}

	var ops []msgs.Op

	// For indices that exist in both lists, generate SET if different
	minLen := fromLen
	if toLen < minLen {
		minLen = toLen
	}
	for i := 0; i < minLen; i++ {
		if fromSlice[i] != toSlice[i] {
			op := msgs.NewOp(nil)
			op.SetFieldNum(fieldNum)
			op.SetType(msgs.ListSet)
			op.SetIndex(int32(i))
			op.SetData(encode(toSlice[i]))
			ops = append(ops, op)
		}
	}

	// For new indices in 'to', generate INSERT
	for i := fromLen; i < toLen; i++ {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListInsert)
		op.SetIndex(int32(i))
		op.SetData(encode(toSlice[i]))
		ops = append(ops, op)
	}

	// For removed indices, generate REMOVE (in reverse order)
	for i := fromLen - 1; i >= toLen; i-- {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListRemove)
		op.SetIndex(int32(i))
		ops = append(ops, op)
	}

	// Use LIST_REPLACE when individual ops exceed half the combined list sizes.
	// This heuristic balances patch size vs granularity of changes.
	if len(ops) > 0 && len(ops) > (fromLen+toLen)/2 {
		replaceOp, err := diffListNumberReplace(from, to, fieldNum, fd)
		if err != nil {
			return nil, err
		}
		if replaceOp != nil {
			return []msgs.Op{*replaceOp}, nil
		}
		return nil, nil
	}

	return ops, nil
}

func diffListNumberFloat[F float32 | float64](from, to *segment.Struct, fieldNum uint16, fd *mapping.FieldDescr, encode func(F) []byte) ([]msgs.Op, error) {
	fromList := segment.GetListNumbers[F](from, fieldNum)
	toList := segment.GetListNumbers[F](to, fieldNum)

	var fromSlice, toSlice []F
	if fromList != nil {
		fromSlice = fromList.Slice()
	}
	if toList != nil {
		toSlice = toList.Slice()
	}

	fromLen := len(fromSlice)
	toLen := len(toSlice)

	if fromLen == 0 && toLen == 0 {
		return nil, nil
	}

	var ops []msgs.Op

	// For indices that exist in both lists, generate SET if different (handling NaN)
	minLen := fromLen
	if toLen < minLen {
		minLen = toLen
	}
	for i := 0; i < minLen; i++ {
		fromNaN := math.IsNaN(float64(fromSlice[i]))
		toNaN := math.IsNaN(float64(toSlice[i]))
		equal := (fromNaN && toNaN) || (!fromNaN && !toNaN && fromSlice[i] == toSlice[i])
		if !equal {
			op := msgs.NewOp(nil)
			op.SetFieldNum(fieldNum)
			op.SetType(msgs.ListSet)
			op.SetIndex(int32(i))
			op.SetData(encode(toSlice[i]))
			ops = append(ops, op)
		}
	}

	// For new indices in 'to', generate INSERT
	for i := fromLen; i < toLen; i++ {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListInsert)
		op.SetIndex(int32(i))
		op.SetData(encode(toSlice[i]))
		ops = append(ops, op)
	}

	// For removed indices, generate REMOVE (in reverse order)
	for i := fromLen - 1; i >= toLen; i-- {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListRemove)
		op.SetIndex(int32(i))
		ops = append(ops, op)
	}

	// Use LIST_REPLACE when individual ops exceed half the combined list sizes.
	// This heuristic balances patch size vs granularity of changes.
	if len(ops) > 0 && len(ops) > (fromLen+toLen)/2 {
		replaceOp, err := diffListNumberReplace(from, to, fieldNum, fd)
		if err != nil {
			return nil, err
		}
		if replaceOp != nil {
			return []msgs.Op{*replaceOp}, nil
		}
		return nil, nil
	}

	return ops, nil
}

func diffListBytes(from, to *segment.Struct, fieldNum uint16, ft field.Type) ([]msgs.Op, error) {
	var fromList, toList interface {
		Len() int
		Get(int) []byte
	}

	if ft == field.FTListStrings {
		fromStrs := segment.GetListStrings(from, fieldNum)
		toStrs := segment.GetListStrings(to, fieldNum)
		if fromStrs != nil {
			fromList = stringsAsBytesGetter{fromStrs}
		}
		if toStrs != nil {
			toList = stringsAsBytesGetter{toStrs}
		}
	} else {
		fromList = segment.GetListBytes(from, fieldNum)
		toList = segment.GetListBytes(to, fieldNum)
	}

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

	var ops []msgs.Op

	// For indices that exist in both lists, generate SET if different
	minLen := fromLen
	if toLen < minLen {
		minLen = toLen
	}
	for i := 0; i < minLen; i++ {
		if !bytes.Equal(fromList.Get(i), toList.Get(i)) {
			op := msgs.NewOp(nil)
			op.SetFieldNum(fieldNum)
			op.SetType(msgs.ListSet)
			op.SetIndex(int32(i))
			op.SetData(toList.Get(i))
			ops = append(ops, op)
		}
	}

	// For new indices in 'to', generate INSERT
	for i := fromLen; i < toLen; i++ {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListInsert)
		op.SetIndex(int32(i))
		op.SetData(toList.Get(i))
		ops = append(ops, op)
	}

	// For removed indices, generate REMOVE (in reverse order)
	for i := fromLen - 1; i >= toLen; i-- {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListRemove)
		op.SetIndex(int32(i))
		ops = append(ops, op)
	}

	// Use LIST_REPLACE when individual ops exceed half the combined list sizes.
	// This heuristic balances patch size vs granularity of changes.
	if len(ops) > 0 && len(ops) > (fromLen+toLen)/2 {
		replaceOp, err := createListReplaceBytes(fieldNum, toList)
		if err != nil {
			return nil, err
		}
		if replaceOp != nil {
			return []msgs.Op{*replaceOp}, nil
		}
		return nil, nil
	}

	return ops, nil
}

// stringsAsBytesGetter adapts a Strings list to return []byte
type stringsAsBytesGetter struct {
	s *segment.Strings
}

func (g stringsAsBytesGetter) Len() int {
	return g.s.Len()
}

func (g stringsAsBytesGetter) Get(i int) []byte {
	return []byte(g.s.Get(i))
}

func diffListStructs(ctx context.Context, from, to *segment.Struct, fieldNum uint16, fd *mapping.FieldDescr) ([]msgs.Op, error) {
	fromList := segment.GetListStructs(ctx, from, fieldNum, fd.Mapping)
	toList := segment.GetListStructs(ctx, to, fieldNum, fd.Mapping)

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

	var ops []msgs.Op

	// For indices that exist in both lists, generate LIST_STRUCT_PATCH if different
	minLen := fromLen
	if toLen < minLen {
		minLen = toLen
	}
	for i := 0; i < minLen; i++ {
		fromItem := fromList.Get(i)
		toItem := toList.Get(i)

		// Compute recursive patch - if structs are equal, this produces 0 ops
		subPatch := msgs.NewPatch(nil)
		subPatch.SetVersion(PatchVersion)
		if err := diffStructToPatch(ctx, fromItem, toItem, &subPatch); err != nil {
			return nil, fmt.Errorf("diff struct at index %d: %w", i, err)
		}

		if subPatch.OpsLen(ctx) > 0 {
			patchBytes, err := subPatch.Marshal()
			if err != nil {
				return nil, fmt.Errorf("marshal sub-patch at index %d: %w", i, err)
			}

			op := msgs.NewOp(nil)
			op.SetFieldNum(fieldNum)
			op.SetType(msgs.ListStructPatch)
			op.SetIndex(int32(i))
			op.SetData(patchBytes)
			ops = append(ops, op)
		}
	}

	// For new indices in 'to', generate INSERT with full struct data
	for i := fromLen; i < toLen; i++ {
		toItem := toList.Get(i)
		buf, err := toItem.Marshal()
		if err != nil {
			return nil, fmt.Errorf("marshal struct for insert at index %d: %w", i, err)
		}

		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListInsert)
		op.SetIndex(int32(i))
		op.SetData(buf)
		ops = append(ops, op)
	}

	// For removed indices, generate REMOVE (in reverse order)
	for i := fromLen - 1; i >= toLen; i-- {
		op := msgs.NewOp(nil)
		op.SetFieldNum(fieldNum)
		op.SetType(msgs.ListRemove)
		op.SetIndex(int32(i))
		ops = append(ops, op)
	}

	// Use LIST_REPLACE when individual ops exceed half the combined list sizes.
	// This heuristic balances patch size vs granularity of changes.
	if len(ops) > 0 && len(ops) > (fromLen+toLen)/2 {
		replaceOp, err := createListReplaceStructs(fieldNum, toList)
		if err != nil {
			return nil, err
		}
		if replaceOp != nil {
			return []msgs.Op{*replaceOp}, nil
		}
		return nil, nil
	}

	return ops, nil
}

// Helper functions for list replace operations
func createListReplaceBools(fieldNum uint16, toList *segment.Bools) (*msgs.Op, error) {
	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(NoListIndex)

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

func diffListNumberReplace(from, to *segment.Struct, fieldNum uint16, fd *mapping.FieldDescr) (*msgs.Op, error) {
	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(NoListIndex)

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

func encodeListNumber[N segment.Number](from, to *segment.Struct, fieldNum uint16, sizeInBytes int) ([]byte, bool) {
	fromList := segment.GetListNumbers[N](from, fieldNum)
	toList := segment.GetListNumbers[N](to, fieldNum)

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

func encodeListNumberFloat32(from, to *segment.Struct, fieldNum uint16) ([]byte, bool) {
	fromList := segment.GetListNumbers[float32](from, fieldNum)
	toList := segment.GetListNumbers[float32](to, fieldNum)

	var fromSlice, toSlice []float32
	if fromList != nil {
		fromSlice = fromList.Slice()
	}
	if toList != nil {
		toSlice = toList.Slice()
	}

	// Check if lists are equal (handling NaN: both NaN is considered equal)
	if len(fromSlice) == len(toSlice) {
		equal := true
		for i := range fromSlice {
			fromNaN := math.IsNaN(float64(fromSlice[i]))
			toNaN := math.IsNaN(float64(toSlice[i]))
			if !((fromNaN && toNaN) || (!fromNaN && !toNaN && fromSlice[i] == toSlice[i])) {
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

func encodeListNumberFloat64(from, to *segment.Struct, fieldNum uint16) ([]byte, bool) {
	fromList := segment.GetListNumbers[float64](from, fieldNum)
	toList := segment.GetListNumbers[float64](to, fieldNum)

	var fromSlice, toSlice []float64
	if fromList != nil {
		fromSlice = fromList.Slice()
	}
	if toList != nil {
		toSlice = toList.Slice()
	}

	// Check if lists are equal (handling NaN: both NaN is considered equal)
	if len(fromSlice) == len(toSlice) {
		equal := true
		for i := range fromSlice {
			fromNaN := math.IsNaN(fromSlice[i])
			toNaN := math.IsNaN(toSlice[i])
			if !((fromNaN && toNaN) || (!fromNaN && !toNaN && fromSlice[i] == toSlice[i])) {
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

func createListReplaceBytes(fieldNum uint16, toList interface {
	Len() int
	Get(int) []byte
}) (*msgs.Op, error) {
	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(NoListIndex)

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

func createListReplaceStructs(fieldNum uint16, toList *segment.Structs) (*msgs.Op, error) {
	op := msgs.NewOp(nil)
	op.SetFieldNum(fieldNum)
	op.SetType(msgs.ListReplace)
	op.SetIndex(NoListIndex)

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
		if _, err := item.MarshalWriter(buf); err != nil {
			return nil, fmt.Errorf("marshal struct at index %d: %w", i, err)
		}
	}

	op.SetData(buf.Bytes())
	return &op, nil
}

// IsEmpty returns true if the patch has no operations.
func IsEmpty(ctx context.Context, p msgs.Patch) bool {
	return p.OpsLen(ctx) == 0
}
