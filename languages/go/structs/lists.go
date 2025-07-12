package structs

import (
	"bytes"
	stdbinary "encoding/binary"
	"fmt"
	"io"
	"iter"
	"log"
	"math"
	"slices"
	"sync/atomic"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/conversions"
	"github.com/bearlytools/claw/internal/typedetect"
	"github.com/bearlytools/claw/languages/go/field"
	"github.com/bearlytools/claw/languages/go/mapping"
	"github.com/bearlytools/claw/languages/go/structs/header"
	"github.com/gostdlib/base/context"
)


// Bools is a wrapper around a list of boolean values.
type Bools struct {
	data []byte // Includes the header
	len  int

	s *Struct
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
	data        []byte
	sizeInBytes uint8 // 1, 2, 3, 4
	len         int
	isFloat     bool

	s *Struct
}

// NewNumbers is used to create a holder for a list of numbers not decoded from an existing []byte stream.
func NewNumbers[I typedetect.Number]() *Numbers[I] {
	var t I
	size := unsafe.Sizeof(t)
	
	var n *Numbers[I]
	var sizeInBytes uint8
	var isFloatType bool
	var ft field.Type
	
	// Determine characteristics using unsafe helpers
	isFloatType = typedetect.IsFloat[I]()
	isSigned := typedetect.IsSignedInteger[I]()
	
	// Create new instance directly (pools don't work with custom types)
	n = &Numbers[I]{}
	
	switch size {
	case 1:
		if isSigned {
			ft = field.FTListInt8
		} else {
			ft = field.FTListUint8
		}
		sizeInBytes = 1
	case 2:
		if isSigned {
			ft = field.FTListInt16
		} else {
			ft = field.FTListUint16
		}
		sizeInBytes = 2
	case 4:
		if isFloatType {
			ft = field.FTListFloat32
		} else if isSigned {
			ft = field.FTListInt32
		} else {
			ft = field.FTListUint32
		}
		sizeInBytes = 4
	case 8:
		if isFloatType {
			ft = field.FTListFloat64
		} else if isSigned {
			ft = field.FTListInt64
		} else {
			ft = field.FTListUint64
		}
		sizeInBytes = 8
	default:
		panic(fmt.Sprintf("unsupported number type %T (size: %d bytes)", t, size))
	}
	
	h := NewGenericHeader()
	h.SetFieldType(ft)
	
	n.sizeInBytes = sizeInBytes
	n.isFloat = isFloatType
	n.data = h
	
	return n
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
	
	// Create new instance directly (pools don't work with custom types)
	n = &Numbers[I]{}
	
	switch size {
	case 1:
		sizeInBytes = 1
	case 2:
		sizeInBytes = 2
	case 4:
		sizeInBytes = 4
	case 8:
		sizeInBytes = 8
	default:
		panic(fmt.Sprintf("unsupported number type %T (size: %d bytes)", t, size))
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
		return
	case 2:
		binary.Put(holder, uint16(value))
		return
	case 4:
		if n.isFloat {
			u := math.Float32bits(float32(value))
			binary.Put(holder, u)
			return
		}
		binary.Put(holder, uint32(value))
		return
	case 8:
		if n.isFloat {
			u := math.Float64bits(float64(value))
			binary.Put(holder, u)
			return
		}
		binary.Put(holder, uint64(value))
		return
	}

	panic("should never get here")
}

// Append appends values to the list of numbers.
func (n *Numbers[I]) Append(i ...I) {
	oldSize := len(n.data)
	defer func() {
		updateItems(n.data[:8], n.len)
		if n.s != nil {
			XXXAddToTotal(n.s, len(n.data)-oldSize)
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

// Bytes represents a list of bytes.
type Bytes struct {
	header GenericHeader
	data   [][]byte // Each entry includes the item header of 32bits.

	s        *Struct
	dataSize int64 // This is the size of the "data" field (without header)
	padding  int64 // This is how much padding would currently be needed
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

	if b.header.Final40() == 0 {
		return nil, fmt.Errorf("cannot have a ListBytes field that has zero entries")
	}

	// We need to carve up the slice into a slice of slice.
	d := make([][]byte, b.header.Final40())

	read := 8 // This will hold the number of bytes we have read.
	for i := 0; i < len(d); i++ {
		if len(*data) < 4 {
			return nil, fmt.Errorf("malformed list of bytes field: an item (%d) did not have a valid header", i)
		}
		size := int(binary.Get[uint32]((*data)[:4]))
		if len((*data)[4:]) < size {
			return nil, fmt.Errorf("malformed list of bytes field: an item did not have enough data to match the header")
		}
		// Assign data
		d[i] = (*data)[:size+4] // data size + data header

		// Move to next set of data
		*data = (*data)[4+size:] // Move past item
		read += size + 4
	}

	// Read past any padding that was required to align to 64 bits (8 bytes).
	paddingNeeded := PaddingNeeded(read)
	if paddingNeeded != 0 {
		if len(*data) < paddingNeeded {
			return nil, fmt.Errorf("malformed list of bytes field: was missing byte list padding")
		}
		*data = (*data)[paddingNeeded:]
		read += paddingNeeded
	}

	log.Println("1: ", read)
	XXXAddToTotal(s, read) // Add header + data + padding

	b.data = d
	b.s = s
	atomic.StoreInt64(&b.dataSize, int64(read-8-paddingNeeded)) // We do not count the list header or padding in this
	atomic.StoreInt64(&b.padding, int64(paddingNeeded))

	return b, nil
}

// Len returns the number of items in the list.
func (b *Bytes) Len() int {
	return len(b.data)
}

// Get gets a []byte stored at the index.
func (b *Bytes) Get(index int) []byte {
	if index >= b.Len() {
		panic(fmt.Sprintf("slice out of bounds: index %d in slice of size %d", index, b.Len()))
	}

	if len(b.data[index]) == 4 {
		return nil
	}

	return b.data[index][4:]
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
	// Record the current size of this value and end padding.  Get new value size and new
	// padding needed. Calculate our new data size.
	oldSize := int64(len(b.data[index]))
	oldPadding := b.padding
	XXXAddToTotal(b.s, -(oldSize + oldPadding))
	atomic.AddInt64(&b.dataSize, -oldSize)

	atomic.AddInt64(&b.dataSize, int64(len(value))+4) // data + entry header
	atomic.StoreInt64(&b.padding, PaddingNeeded(b.dataSize))

	b.set(index, value)
}

func (b *Bytes) set(index int, value []byte) {
	buff := make([]byte, 4+len(value))
	binary.Put(buff, uint32(len(value)))
	copy(buff[4:], value)
	b.data[index] = buff
}

// Append appends values to the list of []byte.
func (b *Bytes) Append(values ...[]byte) {
	for _, v := range values {
		if len(v) > math.MaxUint32 {
			panic(fmt.Sprintf("cannot set a value > %dKiB", math.MaxUint32/1024))
		}
	}

	// Remove old data size = padding.
	if b.s != nil {
		log.Println("removing old data size before append: ", -(b.dataSize + b.padding))
		XXXAddToTotal(b.s, -(b.dataSize + b.padding))
	}

	newSize := b.dataSize // We are appending, so our new size starts at the old size

	// Create new slice that can hold our data.
	indexStart := len(b.data)
	n := make([][]byte, len(b.data)+len(values))
	copy(n, b.data)
	b.data = n

	for i, v := range values {
		b.set(indexStart+i, v)
		newSize += int64(len(v)) + 4 // data + entry header
	}
	updateItems(b.header, len(b.data))

	// Record our data size and padding requirements.
	b.dataSize = newSize
	b.padding = PaddingNeeded(newSize)
	if b.s != nil {
		log.Println("adding new append data size: ", b.dataSize+b.padding)
		XXXAddToTotal(b.s, b.dataSize+b.padding) // data size + entry header size
	}
}

// Slice converts this into a standard [][]byte. The values aren't linked, so changing
// []bool or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (b *Bytes) Slice() [][]byte {
	if len(b.data) == 0 {
		return nil
	}

	n := make([][]byte, len(b.data))
	for i, v := range b.data {
		n[i] = make([]byte, len(v))
		copy(n[i], v)
	}
	return n
}

// Encode returns the []byte to write to output to represent this Bytes. If it returns nil,
// no output should be written.
func (b *Bytes) Encode(w io.Writer) (int, error) {
	// If we have a Bytes that doesn't actually have any data, it should not be encoded as
	// indicated by returning nil.
	if len(b.data) == 0 {
		return 0, nil
	}

	wrote, err := w.Write(b.header)
	if err != nil {
		return wrote, err
	}
	for _, item := range b.data {
		n, err := w.Write(item)
		wrote += n
		if err != nil {
			return wrote, err
		}
	}
	n, err := w.Write(Padding(int(b.padding)))
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

// Structs represents a list of Struct.
type Structs struct {
	header              header.Generic
	data                []*Struct
	mapping             *mapping.Map
	s                   *Struct
	zeroTypeCompression bool
	size                *int64 // The size of the header + all structs in the list.
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
		size:    new(int64),
	}

	s.header.SetFieldNum(0)
	s.header.SetFieldType(field.FTListStructs)
	s.header.SetFinal40(0)
	atomic.AddInt64(s.size, 8)
	return s
}

// NewStructsFromBytes returns a new Bytes value.
func NewStructsFromBytes(data *[]byte, s *Struct, m *mapping.Map) (*Structs, error) {
	if m == nil {
		panic("bug: cannot pass nil *mapping.Map")
	}
	if s == nil {
		panic("bug: cannot pass *Struct == nil")
	}
	// This is an error, because if they want to encode an empty list, it should not get encoded on the
	// wire. There is no need to distinguish a zero value on a list type from not being set.
	if len(*data) < 16 { // structs header(8) + 8 bytes of some field
		return nil, fmt.Errorf("malformed list of structs: must be at least 16 bytes in size")
	}
	d := &Structs{s: s, mapping: m, size: new(int64)}
	d.header = (*data)[:8]
	*data = (*data)[8:] // Move past the header

	if d.header.Final40() == 0 {
		return nil, fmt.Errorf("cannot have a ListStructs field that has zero entries")
	}
	d.data = make([]*Struct, d.header.Final40())
	reader := bytes.NewReader(*data)

	read := 8 // This will hold the number of bytes we have read.
	for i := 0; i < len(d.data); i++ {
		if len(*data) < 8 {
			return nil, fmt.Errorf("malformed list of structs field: an item (%d) did not have a valid header", i)
		}

		entry := New(0, m)
		n, err := entry.unmarshal(reader)
		if err != nil {
			return nil, err
		}
		read += n
		d.data[i] = entry
	}

	*data = (*data)[read-8:] // Move past the data (-8 is for the header we alread moved past)
	XXXAddToTotal(s, read)   // Add header + data
	*d.size = int64(read)
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
	s.data = nil
	s.s = nil
	*s.size = 0
}

// Map returns the Map for all entries in this list of Structs.
func (s *Structs) Map() *mapping.Map {
	return s.mapping
}

// Len returns the number of items in the list.
func (s *Structs) Len() int {
	return len(s.data)
}

// Get gets a *Struct stored at the index.
func (s *Structs) Get(index int) *Struct {
	if index >= s.Len() {
		panic(fmt.Sprintf("slice out of bounds: index %d in slice of size %d", index, s.Len()))
	}

	return s.data[index]
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
	if index >= len(s.data) {
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
	s.data[index] = value

	// Remove the size of the current entry.
	old := s.data[index]
	oldSize := atomic.LoadInt64(old.structTotal)
	XXXAddToTotal(s.s, -oldSize)
	atomic.AddInt64(s.size, -oldSize)

	// Add the new size.
	newSize := atomic.LoadInt64(value.structTotal)
	XXXAddToTotal(s.s, newSize)
	atomic.AddInt64(s.size, newSize)
	return nil
}

// Append appends values to the list of []byte.
func (s *Structs) Append(values ...*Struct) error {
	oldSize := atomic.LoadInt64(s.size)

	var total int64
	for i, v := range values {
		if v == nil {
			return fmt.Errorf("entry to Append() cannot be a nil *Struct")
		}
		if v.parent != nil {
			// TODO(jdoak): If this is true, deep clone the Struct and attach the copy.
			return fmt.Errorf("entry %d is attached to another field", i)
		}
		// Update our value's parent.
		v.parent = s.s

		// If the mapping pointers are pointing to the same place, then the Structs aren't the same.
		if v.mapping != s.mapping {
			return fmt.Errorf("you are attempting to set index %d to a Struct with a different type that the list", i)
		}
		v.zeroTypeCompression = s.zeroTypeCompression
		total += atomic.LoadInt64(v.structTotal)
	}
	s.data = append(s.data, values...)

	// Update the total the list sees.
	atomic.AddInt64(s.size, total)
	XXXAddToTotal(s.s, atomic.LoadInt64(s.size)-oldSize)

	updateItems(s.header, len(s.data))
	return nil
}

// Slice converts this into a standard []*Struct.
func (s *Structs) Slice() []*Struct {
	if len(s.data) == 0 {
		return nil
	}
	return s.data
}

// Encode returns the []byte to write to output to represent this Structs. If it returns nil,
// no output should be written.
func (s *Structs) Encode(w io.Writer) (int, error) {
	// If we have a Structs that doesn't actually have any data, it should not be encoded as
	// indicated by returning nil.
	if len(s.data) == 0 {
		return 0, nil
	}

	wrote, err := w.Write(s.header)
	if err != nil {
		return wrote, err
	}
	log.Println("header was: ", wrote)
	for index, item := range s.data {
		item.header.SetFieldNum(uint16(index))
		n, err := item.Marshal(w)
		wrote += n
		log.Println("wrote item: ", n)
		if err != nil {
			return wrote, err
		}
	}
	log.Println("total was: ", wrote)
	return wrote, err
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
