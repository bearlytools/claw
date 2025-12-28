package structs

import (
	"bytes"
	stdbinary "encoding/binary"
	"fmt"
	"io"
	"iter"
	"math"
	"slices"
	"sync/atomic"
	"unsafe"

	"github.com/bearlytools/claw/clawc/internal/binary"
	"github.com/bearlytools/claw/clawc/internal/bits"
	"github.com/bearlytools/claw/clawc/internal/typedetect"
	"github.com/bearlytools/claw/clawc/languages/go/conversions"
	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
	"github.com/bearlytools/claw/clawc/languages/go/structs/header"
	"github.com/gostdlib/base/context"
)

// Bools is a wrapper around a list of boolean values.
type Bools struct {
	s    *Struct
	data []byte // Includes the header
	len  int
}

// NewBools creates a new Bool that will be stored in a Struct field.
func NewBools(fieldNum uint16) *Bools {
	b := boolPool.Get(context.Background())

	h := NewGenericHeader()
	h.SetFieldNum(fieldNum)
	h.SetFieldType(field.FTListBools)

	b.data = h

	return b
}

// NewBoolsFromBytes returns a new Bool value and advances "data" passed the list.
func NewBoolsFromBytes(data *[]byte, s *Struct) (GenericHeader, *Bools, error) {
	l := len(*data)
	if l < 8 {
		return nil, nil, fmt.Errorf("Struct.decodeListBool() header was < 64 bits")
	}

	h := GenericHeader((*data)[:8])
	items := h.Final40()
	if items == 0 {
		return nil, nil, fmt.Errorf("Struct.decodeListBool() header had item count == 0, which is not allowed")
	}

	wordsNeeded := (items / 64) + 1
	if len((*data)[8:]) < int(wordsNeeded)*8 {
		return nil, nil, fmt.Errorf("malformed: list of boolean: header had data size not consistend with message")
	}
	rightBound := (8 * wordsNeeded) + 8
	sl := (*data)[0:rightBound]
	b := boolPool.Get(context.Background())

	b.data = sl
	b.len = int(items)
	b.s = s

	*data = (*data)[rightBound:]
	XXXAddToTotal(s, len(b.data))
	return h, b, nil
}

// Len returns the number of items in this list of bools.
func (b *Bools) Len() int {
	return b.len
}

// Get gets a value in the list[pos].
func (b *Bools) Get(index int) bool {
	data := b.data[8:]

	if index >= b.len {
		panic(fmt.Sprintf("lists.Bool with len %d cannot have position %d set", b.len, index))
	}

	sliceNum := index / 8
	i := binary.Get[uint8](data[sliceNum : sliceNum+1])
	indexInSlice := index - (sliceNum * 8)

	return bits.GetBit(i, uint8(indexInSlice))
}

// All returns an iterator over all boolean values in the list.
func (b *Bools) All() iter.Seq[bool] {
	return b.Range(0, b.len)
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (b *Bools) Range(from, to int) iter.Seq[bool] {
	return func(yield func(bool) bool) {
		if b.len == 0 {
			return
		}
		if from > b.len-1 {
			panic("Range 'from' argument is out of bounds")
		}
		if to > b.len {
			panic("Range 'to' is out of bounds")
		}
		if from >= to {
			panic("Range 'to' cannot be >= to 'from'")
		}

		for index := from; index < to; index++ {
			if !yield(b.Get(index)) {
				return
			}
		}
	}
}

// Set a boolean in position "pos" to "val".
func (b *Bools) Set(index int, val bool) {
	data := b.data[8:]

	if index >= b.len {
		panic(fmt.Sprintf("lists.Bool with size %d cannot have position %d set", b.len, index))
	}

	// We pack bools into int64s. So 64 bools per 8 bytes.
	sliceNum := index / 8
	i := data[sliceNum]
	indexInSlice := index - (sliceNum * 8) // Now find the bit in the int64 that holds the value

	// Modify the bits and set it.
	i = bits.SetBit(i, uint8(indexInSlice), val)
	data[sliceNum] = i

	// Propagate modified flag to parent
	if b.s != nil {
		b.s.markModified()
	}
}

func (b *Bools) cap() int {
	return (len(b.data) - 8) * 8 // number of bytes * 8 bit values we can hold, minus the header because we don't store there
}

// Append appends values to the list of bools.
func (b *Bools) Append(i ...bool) {
	oldSize := len(b.data)

	requiredCap := b.len + len(i) // in bits
	// If we don't have enough existing capacity to hold the values, extend our
	// capacity. We always extend capacity so the amount is divisible by 64 bits (or 8 bytes).
	if requiredCap > b.cap() {
		wordsNeeded := requiredCap / 64
		if requiredCap%64 != 0 {
			wordsNeeded++
		}
		n := make([]byte, wordsNeeded*8+8) // header + data size needed
		copy(n, b.data)
		b.data = n
	}
	start := b.len
	b.len += len(i)

	for index, val := range i {
		b.Set(start+index, val)
	}

	updateItems(b.data[:8], b.len)
	if b.s != nil {
		XXXAddToTotal(b.s, len(b.data)-oldSize)
		b.s.markModified()
	}
}

// Slice converts this into a standard []bool. The values aren't linked, so changing
// []bool or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (b *Bools) Slice() []bool {
	if b.len == 0 {
		return nil
	}
	return slices.Collect(b.All())
}

// Encode returns the []byte to write to output to represent this Bool. If it returns nil,
// no output should be written.
func (b *Bools) Encode() []byte {
	if b.data == nil {
		return nil
	}
	return b.data
}

// Numbers represents a list of numbers
type Numbers[I typedetect.Number] struct {
	s           *Struct
	data        []byte
	len         int
	sizeInBytes uint8 // 1, 2, 3, 4
	isFloat     bool
}

// NewNumbers is used to create a holder for a list of numbers not decoded from an existing []byte stream.
func NewNumbers[I typedetect.Number]() *Numbers[I] {
	ctx := context.Background()
	var t I
	size := unsafe.Sizeof(t)

	var n *Numbers[I]
	var sizeInBytes uint8
	var isFloatType bool
	var ft field.Type

	// Determine characteristics using unsafe helpers
	isFloatType = typedetect.IsFloat[I]()
	isSigned := typedetect.IsSignedInteger[I]()

	// Try to get from appropriate pool based on type.
	// Use type assertion to check if pooled type matches - it won't for enum types.
	var pooled any
	var ok bool
	switch size {
	case 1:
		if isSigned {
			ft = field.FTListInt8
			pooled = nInt8Pool.Get(ctx)
		} else {
			ft = field.FTListUint8
			pooled = nUint8Pool.Get(ctx)
		}
		sizeInBytes = 1
	case 2:
		if isSigned {
			ft = field.FTListInt16
			pooled = nInt16Pool.Get(ctx)
		} else {
			ft = field.FTListUint16
			pooled = nUint16Pool.Get(ctx)
		}
		sizeInBytes = 2
	case 4:
		if isFloatType {
			ft = field.FTListFloat32
			pooled = nFloat32Pool.Get(ctx)
		} else if isSigned {
			ft = field.FTListInt32
			pooled = nInt32Pool.Get(ctx)
		} else {
			ft = field.FTListUint32
			pooled = nUint32Pool.Get(ctx)
		}
		sizeInBytes = 4
	case 8:
		if isFloatType {
			ft = field.FTListFloat64
			pooled = nFloat64Pool.Get(ctx)
		} else if isSigned {
			ft = field.FTListInt64
			pooled = nInt64Pool.Get(ctx)
		} else {
			ft = field.FTListUint64
			pooled = nUint64Pool.Get(ctx)
		}
		sizeInBytes = 8
	default:
		panic(fmt.Sprintf("unsupported number type %T (size: %d bytes)", t, size))
	}

	// Type assertion - if I is an enum type, this will fail and we create new one.
	n, ok = pooled.(*Numbers[I])
	if !ok {
		// Return pooled value since type doesn't match (e.g., enum types)
		returnNumberToPool(ctx, pooled, sizeInBytes, isFloatType, isSigned)
		n = &Numbers[I]{}
	}

	h := NewGenericHeader()
	h.SetFieldType(ft)

	n.sizeInBytes = sizeInBytes
	n.isFloat = isFloatType
	n.data = h
	n.len = 0
	n.s = nil

	return n
}

// returnNumberToPool returns a pooled Numbers value back to its pool.
func returnNumberToPool(ctx context.Context, pooled any, sizeInBytes uint8, isFloat, isSigned bool) {
	switch sizeInBytes {
	case 1:
		if isSigned {
			nInt8Pool.Put(ctx, pooled.(*Numbers[int8]))
		} else {
			nUint8Pool.Put(ctx, pooled.(*Numbers[uint8]))
		}
	case 2:
		if isSigned {
			nInt16Pool.Put(ctx, pooled.(*Numbers[int16]))
		} else {
			nUint16Pool.Put(ctx, pooled.(*Numbers[uint16]))
		}
	case 4:
		if isFloat {
			nFloat32Pool.Put(ctx, pooled.(*Numbers[float32]))
		} else if isSigned {
			nInt32Pool.Put(ctx, pooled.(*Numbers[int32]))
		} else {
			nUint32Pool.Put(ctx, pooled.(*Numbers[uint32]))
		}
	case 8:
		if isFloat {
			nFloat64Pool.Put(ctx, pooled.(*Numbers[float64]))
		} else if isSigned {
			nInt64Pool.Put(ctx, pooled.(*Numbers[int64]))
		} else {
			nUint64Pool.Put(ctx, pooled.(*Numbers[uint64]))
		}
	}
}

func wordsRequiredToStore(items, sizeInBytes int) int {
	required := (sizeInBytes * items)
	words := required / 8
	if required%8 > 0 {
		words++
	}
	return words
}

// NewNumbersFromBytes returns a new Number value.
func NewNumbersFromBytes[I typedetect.Number](data *[]byte, s *Struct) (*Numbers[I], error) {
	ctx := context.Background()
	l := len(*data)
	if l < 8 {
		return nil, fmt.Errorf("header was < 64 bits")
	}

	h := GenericHeader((*data)[:8])
	items := h.Final40()
	if items == 0 {
		return nil, fmt.Errorf("list of Numbers had zero items, which is an encoding error")
	}

	var t I
	size := unsafe.Sizeof(t)

	var n *Numbers[I]
	var sizeInBytes uint8
	var isFloatType bool

	// Determine characteristics using unsafe helpers
	isFloatType = typedetect.IsFloat[I]()
	isSigned := typedetect.IsSignedInteger[I]()

	// Try to get from appropriate pool based on type
	var pooled any
	var ok bool
	switch size {
	case 1:
		sizeInBytes = 1
		if isSigned {
			pooled = nInt8Pool.Get(ctx)
		} else {
			pooled = nUint8Pool.Get(ctx)
		}
	case 2:
		sizeInBytes = 2
		if isSigned {
			pooled = nInt16Pool.Get(ctx)
		} else {
			pooled = nUint16Pool.Get(ctx)
		}
	case 4:
		sizeInBytes = 4
		if isFloatType {
			pooled = nFloat32Pool.Get(ctx)
		} else if isSigned {
			pooled = nInt32Pool.Get(ctx)
		} else {
			pooled = nUint32Pool.Get(ctx)
		}
	case 8:
		sizeInBytes = 8
		if isFloatType {
			pooled = nFloat64Pool.Get(ctx)
		} else if isSigned {
			pooled = nInt64Pool.Get(ctx)
		} else {
			pooled = nUint64Pool.Get(ctx)
		}
	default:
		panic(fmt.Sprintf("unsupported number type %T (size: %d bytes)", t, size))
	}

	// Type assertion - if I is an enum type, this will fail and we create new
	n, ok = pooled.(*Numbers[I])
	if !ok {
		// Return pooled value since type doesn't match (e.g., enum types)
		returnNumberToPool(ctx, pooled, sizeInBytes, isFloatType, isSigned)
		n = &Numbers[I]{}
	}
	n.sizeInBytes = sizeInBytes
	n.isFloat = isFloatType

	requiredWords := wordsRequiredToStore(int(items), int(n.sizeInBytes))

	if len((*data)[8:]) < int(requiredWords)*8 {
		return nil, fmt.Errorf("malformed: list of numbers[%d bits]: header had data size not consistend with message", sizeInBytes)
	}

	rightBound := (8 * requiredWords) + 8 // datasize(8 * requiredWords) + header(8)
	n.data = (*data)[0:rightBound]
	n.len = int(items)
	n.s = s
	XXXAddToTotal(s, len(n.data))

	// Advance the slice.
	*data = (*data)[rightBound:]

	return n, nil
}

// Len returns the number of items in this list.
func (n *Numbers[I]) Len() int {
	return n.len
}

// Get gets a number stored at the index.
func (n *Numbers[I]) Get(index int) I {
	data := n.data[8:]

	if index >= n.len {
		panic(fmt.Sprintf("lists.Number with len %d cannot get position %d", n.len, index))
	}

	start := index * int(n.sizeInBytes)

	holder := data[start : start+int(n.sizeInBytes)]
	switch n.sizeInBytes {
	case 1:
		u := binary.Get[uint8](holder)
		return I(u)
	case 2:
		u := binary.Get[uint16](holder)
		return I(u)
	case 4:
		if n.isFloat {
			u := stdbinary.LittleEndian.Uint32(holder)
			return I(math.Float32frombits(u))
		}
		u := binary.Get[uint32](holder)
		return I(u)
	case 8:
		if n.isFloat {
			u := stdbinary.LittleEndian.Uint64(holder)
			return I(math.Float64frombits(u))
		}
		u := binary.Get[uint64](holder)
		return I(u)
	}

	panic("should never get here")
}

// All returns an iterator over all numeric values in the list.
func (n *Numbers[I]) All() iter.Seq[I] {
	return n.Range(0, n.len)
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (n *Numbers[I]) Range(from, to int) iter.Seq[I] {
	return func(yield func(I) bool) {
		if n.len == 0 {
			return
		}
		if from > n.len-1 {
			panic("Range 'from' argument is out of bounds")
		}
		if to > n.len {
			panic("Range 'to' is out of bounds")
		}
		if from >= to {
			panic("Range 'to' cannot be >= to 'from'")
		}

		for index := from; index < to; index++ {
			if !yield(n.Get(index)) {
				return
			}
		}
	}
}

// Set a number in position "index" to "value".
func (n *Numbers[I]) Set(index int, value I) {
	data := n.data[8:]

	if index >= n.len {
		panic(fmt.Sprintf("lists.Number with len %d cannot have position %d set", n.len, index))
	}

	start := index * int(n.sizeInBytes)

	holder := data[start : start+int(n.sizeInBytes)]
	switch n.sizeInBytes {
	case 1:
		binary.Put(holder, uint8(value))
	case 2:
		binary.Put(holder, uint16(value))
	case 4:
		if n.isFloat {
			u := math.Float32bits(float32(value))
			binary.Put(holder, u)
		} else {
			binary.Put(holder, uint32(value))
		}
	case 8:
		if n.isFloat {
			u := math.Float64bits(float64(value))
			binary.Put(holder, u)
		} else {
			binary.Put(holder, uint64(value))
		}
	default:
		panic("should never get here")
	}

	// Propagate modified flag to parent
	if n.s != nil {
		n.s.markModified()
	}
}

// Append appends values to the list of numbers.
func (n *Numbers[I]) Append(i ...I) {
	oldSize := len(n.data)
	defer func() {
		updateItems(n.data[:8], n.len)
		if n.s != nil {
			XXXAddToTotal(n.s, len(n.data)-oldSize)
			n.s.markModified()
		}
	}()

	requiredWords := wordsRequiredToStore(n.len+len(i), int(n.sizeInBytes))

	c := make([]byte, (requiredWords*8)+8) // +8 is header space
	copy(c, n.data)
	n.data = c

	start := n.len
	n.len += len(i)
	for index, value := range i {
		n.Set(start+index, value)
	}
}

// Slice converts this into a standard []I, where I is a number value. The values aren't linked, so changing
// []I or calling n.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (n *Numbers[I]) Slice() []I {
	if n.len == 0 {
		return nil
	}
	return slices.Collect(n.All())
}

// Encode returns the []byte to write to output to represent this Number. If it returns nil,
// no output should be written.
func (n *Numbers[I]) Encode() []byte {
	if n.data == nil {
		return nil
	}
	return n.data
}

// Bytes represents a list of bytes with contiguous storage for cache efficiency.
// Instead of [][]byte which requires pointer chasing, we store:
// - offsets: index of start position for each item in data (len = num_items + 1)
// - data: single contiguous buffer holding all item data
type Bytes struct {
	s       *Struct
	header  GenericHeader
	offsets []uint32 // offsets[i] = start of item i in data; offsets[len(offsets)-1] = end
	data    []byte   // All item data stored contiguously

	dataSize atomic.Int64 // Wire size of data (item headers + item data, excludes list header)
	padding  atomic.Int64 // Padding needed for 8-byte alignment
}

// NewBytes returns a new Bytes for holding lists of bytes. This is used when creating a new list
// not attached to a Struct yet.
func NewBytes() *Bytes {
	b := bytesPool.Get(context.Background())
	if b.header == nil {
		b.header = NewGenericHeader()
	}
	b.header.SetFieldNum(0)
	b.header.SetFieldType(field.FTListBytes)
	b.header.SetFinal40(0)
	return b
}

// NewBytesFromBytes returns a new Bytes value.
func NewBytesFromBytes(data *[]byte, s *Struct) (*Bytes, error) {
	// This is an error, because if they want to encode an empty list, it should not get encoded on the
	// wire. There is no need to distinguish a zero value on a list type from not being set.
	if len(*data) < 16 { // list header(8) + entry header(4) + at least 4 byte (1 bytes of data + 3 padding)
		return nil, fmt.Errorf("malformed list of bytes: must be at least 16 bytes in size")
	}
	b := bytesPool.Get(context.Background())
	b.header = (*data)[:8]
	*data = (*data)[8:] // Move past the header

	numItems := int(b.header.Final40())
	if numItems == 0 {
		return nil, fmt.Errorf("cannot have a ListBytes field that has zero entries")
	}

	// First pass: calculate total data size
	tempData := *data
	totalDataSize := 0
	for i := 0; i < numItems; i++ {
		if len(tempData) < 4 {
			return nil, fmt.Errorf("malformed list of bytes field: an item (%d) did not have a valid header", i)
		}
		size := int(binary.Get[uint32](tempData[:4]))
		if len(tempData[4:]) < size {
			return nil, fmt.Errorf("malformed list of bytes field: an item did not have enough data to match the header")
		}
		totalDataSize += size
		tempData = tempData[4+size:]
	}

	// Allocate contiguous buffer and offsets array
	b.offsets = make([]uint32, numItems+1)
	b.data = make([]byte, totalDataSize)

	// Second pass: copy data into contiguous buffer
	offset := uint32(0)
	wireDataSize := 0 // Size of item headers + item data on wire
	for i := 0; i < numItems; i++ {
		size := int(binary.Get[uint32]((*data)[:4]))
		*data = (*data)[4:] // Move past item header
		wireDataSize += 4

		b.offsets[i] = offset
		copy(b.data[offset:], (*data)[:size])
		offset += uint32(size)
		*data = (*data)[size:] // Move past item data
		wireDataSize += size
	}
	b.offsets[numItems] = offset // End offset

	// Read past any padding that was required to align to 64 bits (8 bytes).
	read := 8 + wireDataSize // list header + item headers + item data
	paddingNeeded := PaddingNeeded(read)
	if paddingNeeded != 0 {
		if len(*data) < paddingNeeded {
			return nil, fmt.Errorf("malformed list of bytes field: was missing byte list padding")
		}
		*data = (*data)[paddingNeeded:]
	}

	XXXAddToTotal(s, read+paddingNeeded) // Add header + data + padding

	b.s = s
	b.dataSize.Store(int64(wireDataSize)) // item headers + item data (excludes list header)
	b.padding.Store(int64(paddingNeeded))

	return b, nil
}

// Len returns the number of items in the list.
func (b *Bytes) Len() int {
	if len(b.offsets) == 0 {
		return 0
	}
	return len(b.offsets) - 1
}

// Get gets a []byte stored at the index.
func (b *Bytes) Get(index int) []byte {
	if index >= b.Len() {
		panic(fmt.Sprintf("slice out of bounds: index %d in slice of size %d", index, b.Len()))
	}

	start := b.offsets[index]
	end := b.offsets[index+1]
	if start == end {
		return nil // Empty entry
	}
	return b.data[start:end]
}

// All returns an iterator over all byte slices in the list.
// You should NOT modify the returned []byte slices.
func (b *Bytes) All() iter.Seq[[]byte] {
	return b.Range(0, b.Len())
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (b *Bytes) Range(from, to int) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		if b.Len() == 0 {
			return
		}
		if from > b.Len()-1 {
			panic("Range 'from' argument is out of bounds")
		}
		if to > b.Len() {
			panic("Range 'to' is out of bounds")
		}
		if from >= to {
			panic("Range 'to' cannot be >= to 'from'")
		}

		for index := from; index < to; index++ {
			if !yield(b.Get(index)) {
				return
			}
		}
	}
}

// Set a number in position "index" to "value".
func (b *Bytes) Set(index int, value []byte) {
	if index >= b.Len() {
		panic(fmt.Sprintf("slice out of bounds: index %d in slice of size %d", index, b.Len()))
	}
	if len(value) > math.MaxUint32 {
		panic(fmt.Sprintf("cannot set a value > %dKiB", math.MaxUint32/1024))
	}

	// Calculate old and new sizes
	oldStart := b.offsets[index]
	oldEnd := b.offsets[index+1]
	oldSize := int(oldEnd - oldStart)
	newSize := len(value)
	delta := newSize - oldSize

	// Remove old totals from parent struct
	if b.s != nil {
		XXXAddToTotal(b.s, -(b.dataSize.Load() + b.padding.Load()))
	}

	if delta == 0 {
		// Same size: just copy in place
		copy(b.data[oldStart:], value)
	} else {
		// Different size: rebuild data buffer
		newData := make([]byte, len(b.data)+delta)
		copy(newData[:oldStart], b.data[:oldStart])
		copy(newData[oldStart:], value)
		copy(newData[oldStart+uint32(newSize):], b.data[oldEnd:])
		b.data = newData

		// Update offsets for all items after this one
		for i := index + 1; i < len(b.offsets); i++ {
			b.offsets[i] = uint32(int(b.offsets[i]) + delta)
		}
	}

	// Recalculate wire size: item headers (4 bytes each) + item data
	wireSize := int64(b.Len()*4 + len(b.data))
	b.dataSize.Store(wireSize)
	b.padding.Store(PaddingNeeded(8 + wireSize)) // +8 for list header

	// Add new totals to parent struct
	if b.s != nil {
		XXXAddToTotal(b.s, b.dataSize.Load()+b.padding.Load())
		b.s.markModified()
	}
}

// Append appends values to the list of []byte.
func (b *Bytes) Append(values ...[]byte) {
	for _, v := range values {
		if len(v) > math.MaxUint32 {
			panic(fmt.Sprintf("cannot set a value > %dKiB", math.MaxUint32/1024))
		}
	}

	// Remove old data size + padding.
	if b.s != nil {
		XXXAddToTotal(b.s, -(b.dataSize.Load() + b.padding.Load()))
	}

	// Calculate new sizes
	additionalSize := 0
	for _, v := range values {
		additionalSize += len(v)
	}

	// Extend offsets array
	oldLen := b.Len()
	newOffsets := make([]uint32, oldLen+len(values)+1)
	copy(newOffsets, b.offsets)

	// Extend data buffer
	newData := make([]byte, len(b.data)+additionalSize)
	copy(newData, b.data)

	// Append new items
	offset := uint32(len(b.data))
	for i, v := range values {
		newOffsets[oldLen+i] = offset
		copy(newData[offset:], v)
		offset += uint32(len(v))
	}
	newOffsets[oldLen+len(values)] = offset // End offset

	b.offsets = newOffsets
	b.data = newData
	updateItems(b.header, b.Len())

	// Record our data size and padding requirements.
	wireSize := int64(b.Len()*4 + len(b.data)) // item headers + item data
	b.dataSize.Store(wireSize)
	b.padding.Store(PaddingNeeded(8 + wireSize)) // +8 for list header

	if b.s != nil {
		XXXAddToTotal(b.s, b.dataSize.Load()+b.padding.Load())
		b.s.markModified()
	}
}

// Slice converts this into a standard [][]byte. The values aren't linked, so changing
// []bool or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (b *Bytes) Slice() [][]byte {
	if b.Len() == 0 {
		return nil
	}

	// Single allocation for result slice and backing buffer
	result := make([][]byte, b.Len())
	backing := make([]byte, len(b.data))
	copy(backing, b.data)

	// Create subslices pointing into the backing buffer
	for i := 0; i < b.Len(); i++ {
		start := b.offsets[i]
		end := b.offsets[i+1]
		if start == end {
			result[i] = nil // Empty entry
			continue
		}
		result[i] = backing[start:end]
	}
	return result
}

// Encode returns the []byte to write to output to represent this Bytes. If it returns nil,
// no output should be written.
func (b *Bytes) Encode(w io.Writer) (int, error) {
	// If we have a Bytes that doesn't actually have any data, it should not be encoded as
	// indicated by returning nil.
	if b.Len() == 0 {
		return 0, nil
	}

	wrote, err := w.Write(b.header)
	if err != nil {
		return wrote, err
	}

	// Write each item with its 4-byte size header
	var buf [4]byte
	for i := 0; i < b.Len(); i++ {
		itemData := b.Get(i)
		binary.Put(buf[:], uint32(len(itemData)))

		n, err := w.Write(buf[:])
		wrote += n
		if err != nil {
			return wrote, err
		}

		if len(itemData) > 0 {
			n, err = w.Write(itemData)
			wrote += n
			if err != nil {
				return wrote, err
			}
		}
	}

	n, err := w.Write(Padding(int(b.padding.Load())))
	wrote += n
	return wrote, err
}

// Strings represents a list of strings.
type Strings struct {
	l *Bytes
}

// Bytes returns the underlying Bytes implementation. This is for internal use out side of that
// has no support.
func (s Strings) Bytes() *Bytes {
	return s.l
}

// Len returns the number of items in the list.
func (s Strings) Len() int {
	return s.l.Len()
}

// Get gets a string stored at the index.
func (s Strings) Get(index int) string {
	b := s.l.Get(index)
	if b == nil {
		return ""
	}
	return conversions.ByteSlice2String(b)
}

// All returns an iterator over all strings in the list.
func (s Strings) All() iter.Seq[string] {
	return s.Range(0, s.Len())
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (s Strings) Range(from, to int) iter.Seq[string] {
	return func(yield func(string) bool) {
		for b := range s.l.Range(from, to) {
			if !yield(conversions.ByteSlice2String(b)) {
				return
			}
		}
	}
}

// Set a number in position "index" to "value".
func (s Strings) Set(index int, value string) {
	s.l.Set(index, conversions.UnsafeGetBytes(value))
}

// Append appends values to the list of []byte.
func (s Strings) Append(values ...string) {
	x := make([][]byte, len(values))
	for i, v := range values {
		x[i] = conversions.UnsafeGetBytes(v)
	}
	s.l.Append(x...)
}

// Slice converts this into a standard []string. The values aren't linked, so changing
// []string or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (s Strings) Slice() []string {
	if s.l.Len() == 0 {
		return nil
	}
	return slices.Collect(s.All())
}

// Structs represents a list of Struct with contiguous storage and per-item lazy decode.
type Structs struct {
	mapping *mapping.Map
	s       *Struct
	header  header.Generic

	// Contiguous raw storage
	rawData []byte   // All struct bytes stored contiguously
	offsets []uint32 // offsets[i] = start of struct i; offsets[len] = end

	// Lazy decode cache
	decoded []*Struct // nil until Get(i) called
	dirty   []bool    // true if decoded[i] was modified

	size         atomic.Int64 // The size of the header + all structs in the list.
	isSetEnabled bool
}

// NewStructs returns a new Structs for holding lists of Structs. This is used when creating a new list
// not attached to a Struct yet.
func NewStructs(m *mapping.Map) *Structs {
	if m == nil {
		panic("*mapping.map cannot be nil")
	}
	s := &Structs{
		header:  header.New(),
		mapping: m,
	}

	s.header.SetFieldNum(0)
	s.header.SetFieldType(field.FTListStructs)
	s.header.SetFinal40(0)
	s.size.Add(8)
	return s
}

// NewStructsFromBytes returns a new Bytes value.
// s can be nil for lazy decoding (size is already accounted for in parent).
func NewStructsFromBytes(data *[]byte, s *Struct, m *mapping.Map) (*Structs, error) {
	return NewStructsFromBytesWithIsSet(data, s, m, false)
}

// NewStructsFromBytesWithIsSet returns a new Structs value with IsSet propagation.
// s can be nil for lazy decoding (size is already accounted for in parent).
// If isSetEnabled is true, each struct in the list will have IsSet tracking enabled.
// This uses contiguous storage and per-item lazy decode for cache efficiency.
func NewStructsFromBytesWithIsSet(data *[]byte, s *Struct, m *mapping.Map, isSetEnabled bool) (*Structs, error) {
	if m == nil {
		panic("bug: cannot pass nil *mapping.Map")
	}
	// This is an error, because if they want to encode an empty list, it should not get encoded on the
	// wire. There is no need to distinguish a zero value on a list type from not being set.
	if len(*data) < 16 { // structs header(8) + 8 bytes of some field
		return nil, fmt.Errorf("malformed list of structs: must be at least 16 bytes in size")
	}
	d := &Structs{s: s, mapping: m, isSetEnabled: isSetEnabled}
	d.header = (*data)[:8]
	*data = (*data)[8:] // Move past the header

	numItems := int(d.header.Final40())
	if numItems == 0 {
		return nil, fmt.Errorf("cannot have a ListStructs field that has zero entries")
	}

	// First pass: scan to find struct boundaries and total size
	d.offsets = make([]uint32, numItems+1)
	tempData := *data
	offset := uint32(0)

	for i := 0; i < numItems; i++ {
		if len(tempData) < 8 {
			return nil, fmt.Errorf("malformed list of structs field: an item (%d) did not have a valid header", i)
		}
		d.offsets[i] = offset

		// Read struct header to get size
		structHeader := header.Generic(tempData[:8])
		structSize := int(structHeader.Final40())

		if len(tempData) < structSize {
			return nil, fmt.Errorf("malformed list of structs field: item %d claims size %d but only %d bytes remain", i, structSize, len(tempData))
		}

		offset += uint32(structSize)
		tempData = tempData[structSize:]
	}
	d.offsets[numItems] = offset

	// Copy raw data contiguously
	totalSize := int(offset)
	d.rawData = make([]byte, totalSize)
	copy(d.rawData, (*data)[:totalSize])
	*data = (*data)[totalSize:]

	// Initialize lazy decode cache (all nil)
	d.decoded = make([]*Struct, numItems)
	d.dirty = make([]bool, numItems)

	// Size tracking
	read := 8 + totalSize // header + data
	XXXAddToTotal(s, read)
	d.size.Store(int64(read))

	return d, nil
}

// New creates a new *Struct that can be stored in Structs.
func (s *Structs) New() *Struct {
	return New(0, s.mapping)
}

// Reset resets all the internal fields to their zero value. This should only be used
// when recycling the Structs as it does not reset parent size counters.
func (s *Structs) Reset() {
	s.header = nil
	s.rawData = nil
	s.offsets = nil
	s.decoded = nil
	s.dirty = nil
	s.s = nil
	s.size.Store(0)
}

// Map returns the Map for all entries in this list of Structs.
func (s *Structs) Map() *mapping.Map {
	return s.mapping
}

// Len returns the number of items in the list.
func (s *Structs) Len() int {
	if len(s.offsets) == 0 {
		return 0
	}
	return len(s.offsets) - 1
}

// Get gets a *Struct stored at the index. The struct is lazily decoded on first access.
func (s *Structs) Get(index int) *Struct {
	if index >= s.Len() {
		panic(fmt.Sprintf("slice out of bounds: index %d in slice of size %d", index, s.Len()))
	}

	// Return cached if already decoded
	if s.decoded[index] != nil {
		return s.decoded[index]
	}

	// Lazy decode from raw bytes
	start := s.offsets[index]
	end := s.offsets[index+1]
	rawBytes := s.rawData[start:end]

	entry := New(0, s.mapping)
	if s.isSetEnabled {
		entry.XXXSetIsSetEnabled()
	}

	reader := bytes.NewReader(rawBytes)
	if _, err := entry.Unmarshal(reader); err != nil {
		panic(fmt.Sprintf("failed to decode struct %d: %v", index, err))
	}

	// Cache the decoded struct
	s.decoded[index] = entry
	return entry
}

// All returns an iterator over all structs in the list.
func (s *Structs) All() iter.Seq[*Struct] {
	return s.Range(0, s.Len())
}

// Range ranges from "from" (inclusive) to "to" (exclusive).
func (s *Structs) Range(from, to int) iter.Seq[*Struct] {
	return func(yield func(*Struct) bool) {
		if s.Len() == 0 {
			return
		}
		if from > s.Len()-1 {
			panic("Range 'from' argument is out of bounds")
		}
		if to > s.Len() {
			panic("Range 'to' is out of bounds")
		}
		if from >= to {
			panic("Range 'to' cannot be >= to 'from'")
		}

		for index := from; index < to; index++ {
			if !yield(s.Get(index)) {
				return
			}
		}
	}
}

// Set a number in position "index" to "value".
func (s *Structs) Set(index int, value *Struct) error {
	if index >= s.Len() {
		return fmt.Errorf("index %d is not valid", index)
	}

	if value == nil {
		return fmt.Errorf("cannot set the value of a nil *Struct")
	}

	if value.parent != nil {
		return fmt.Errorf("cannot add a *Struct to a list of structs that is attached to another field")
	}

	// If the mapping pointers are not pointing to the same place, then the Structs aren't the same.
	if value.mapping != s.mapping {
		return fmt.Errorf("you are attempting to set index %d to a Struct with a different type that the list", index)
	}

	// Calculate old size (from raw data or decoded struct)
	var oldSize int64
	if s.decoded[index] != nil {
		oldSize = s.decoded[index].structTotal.Load()
	} else {
		// Size from raw data
		oldSize = int64(s.offsets[index+1] - s.offsets[index])
	}
	XXXAddToTotal(s.s, -oldSize)
	s.size.Add(-oldSize)

	// Update cache and mark dirty
	s.decoded[index] = value
	s.dirty[index] = true

	// Add the new size.
	newSize := value.structTotal.Load()
	XXXAddToTotal(s.s, newSize)
	s.size.Add(newSize)

	// Propagate modified flag to parent
	if s.s != nil {
		s.s.markModified()
	}

	return nil
}

// Append appends values to the list of Structs.
func (s *Structs) Append(values ...*Struct) error {
	oldSize := s.size.Load()

	var total int64
	for i, v := range values {
		if v == nil {
			return fmt.Errorf("entry to Append() cannot be a nil *Struct")
		}
		if v.parent != nil {
			// TODO(jdoak): If this is true, deep clone the Struct and attach the copy.
			return fmt.Errorf("entry %d is attached to another field", i)
		}

		// If the mapping pointers are pointing to the same place, then the Structs aren't the same.
		if v.mapping != s.mapping {
			return fmt.Errorf("you are attempting to set index %d to a Struct with a different type that the list", i)
		}
		// Propagate isSetEnabled if the parent has it enabled.
		// IMPORTANT: Do this BEFORE setting parent to avoid double-counting IsSet size
		// (XXXSetIsSetEnabled propagates to parent via XXXAddToTotal).
		if s.isSetEnabled && !v.isSetEnabled {
			v.XXXSetIsSetEnabled()
		}
		total += v.structTotal.Load()

		// Update our value's parent AFTER propagating isSetEnabled.
		v.parent = s.s
	}

	// Extend offsets, decoded, dirty arrays
	oldLen := s.Len()
	newOffsets := make([]uint32, oldLen+len(values)+1)
	copy(newOffsets, s.offsets)

	newDecoded := make([]*Struct, oldLen+len(values))
	copy(newDecoded, s.decoded)

	newDirty := make([]bool, oldLen+len(values))
	copy(newDirty, s.dirty)

	// Append new structs (mark as dirty since they need encoding)
	var currentOffset uint32
	if oldLen > 0 {
		currentOffset = s.offsets[oldLen]
	}
	for i, v := range values {
		newOffsets[oldLen+i] = currentOffset
		newDecoded[oldLen+i] = v
		newDirty[oldLen+i] = true
		// Estimate size for offset tracking (will be exact after encode)
		currentOffset += uint32(v.structTotal.Load())
	}
	newOffsets[oldLen+len(values)] = currentOffset

	s.offsets = newOffsets
	s.decoded = newDecoded
	s.dirty = newDirty

	// Update the total the list sees.
	s.size.Add(total)
	XXXAddToTotal(s.s, s.size.Load()-oldSize)

	updateItems(s.header, s.Len())

	// Propagate modified flag to parent
	if s.s != nil {
		s.s.markModified()
	}

	return nil
}

// Slice converts this into a standard []*Struct.
// This triggers lazy decode for all items.
func (s *Structs) Slice() []*Struct {
	if s.Len() == 0 {
		return nil
	}
	result := make([]*Struct, s.Len())
	for i := 0; i < s.Len(); i++ {
		result[i] = s.Get(i) // Triggers lazy decode if needed
	}
	return result
}

// Encode returns the []byte to write to output to represent this Structs. If it returns nil,
// no output should be written.
func (s *Structs) Encode(w io.Writer) (int, error) {
	// If we have a Structs that doesn't actually have any data, it should not be encoded as
	// indicated by returning nil.
	if s.Len() == 0 {
		return 0, nil
	}

	wrote, err := w.Write(s.header)
	if err != nil {
		return wrote, err
	}

	// Check if any items are dirty
	anyDirty := false
	for _, d := range s.dirty {
		if d {
			anyDirty = true
			break
		}
	}

	// Fast path: no modifications, write raw data directly
	if !anyDirty && s.rawData != nil {
		// Check if any items were decoded (and potentially modified internally)
		allClean := true
		for _, dec := range s.decoded {
			if dec != nil {
				allClean = false
				break
			}
		}
		if allClean {
			n, err := w.Write(s.rawData)
			wrote += n
			return wrote, err
		}
	}

	// Slow path: encode each item
	for i := 0; i < s.Len(); i++ {
		if s.dirty[i] || s.decoded[i] != nil {
			// Encode from decoded struct
			item := s.decoded[i]
			if item == nil {
				// This shouldn't happen if dirty[i] is true, but decode if needed
				item = s.Get(i)
			}
			item.header.SetFieldNum(uint16(i))
			n, err := item.Marshal(w)
			wrote += n
			if err != nil {
				return wrote, err
			}
		} else {
			// Write raw bytes for untouched items
			start := s.offsets[i]
			end := s.offsets[i+1]
			n, err := w.Write(s.rawData[start:end])
			wrote += n
			if err != nil {
				return wrote, err
			}
		}
	}

	return wrote, nil
}

// udpateItems updates list header information to reflect the number items.
func updateItems(b []byte, items int) {
	if items > maxDataSize {
		panic(fmt.Sprintf("cannot add more that %d into a list", maxDataSize))
	}
	// Write to the header our new size.
	u := binary.Get[uint64](b[:8])
	u = bits.SetValue(uint64(items), u, 24, 64)
	binary.Put(b[:8], u)
}
