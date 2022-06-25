package structs

import (
	"context"
	stdbinary "encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"sync/atomic"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
	"golang.org/x/exp/constraints"
)

// Numbers represents all int, uint and float types.
type Numbers interface {
	constraints.Integer | constraints.Float
}

// Bool is a wrapper around a list of boolean values.
type Bool struct {
	data []byte
	len  int

	s *Struct
}

// NewBool creates a new Bool that will be stored in a Struct field with number fieldNum.
func NewBool(fieldNum uint16) *Bool {
	b := pool.Get(boolPool).(*Bool)
	b.data = make([]byte, 8)

	var u uint64
	bits.SetValue(fieldNum, u, 0, 16)
	bits.SetValue(uint8(field.FTListBool), u, 16, 24)
	binary.Put(b.data, u)

	return b
}

// NewBoolFromBytes returns a new Bool value and advances "data" passed the list.
func NewBoolFromBytes(data *[]byte, s *Struct) (*Bool, error) {
	l := len(*data)
	if l < 8 {
		return nil, fmt.Errorf("Struct.decodeListBool() header was < 64 bits")
	}

	i := binary.Get[uint64]((*data)[:8])
	items := bits.GetValue[uint64, uint64](i, dataSizeMask, 24)

	if items == 0 {
		b := pool.Get(boolPool).(*Bool)
		b.data = (*data)[:8]
		b.len = 0
		b.s = s

		*data = (*data)[8:]
		addToTotal(s, 8)
		return b, nil
	}

	wordsNeeded := (items / 64) + 1
	if len((*data)[8:]) < int(wordsNeeded)*8 {
		return nil, fmt.Errorf("malformed: list of boolean: header had data size not consistend with message")
	}
	rightBound := (8 * wordsNeeded) + 8
	sl := (*data)[0:rightBound]
	b := pool.Get(boolPool).(*Bool)

	b.data = sl
	b.len = int(items)
	b.s = s

	*data = (*data)[rightBound:]
	addToTotal(s, len(b.data))
	return b, nil
}

// Len returns the number of items in this list of bools.
func (b *Bool) Len() int {
	return b.len
}

// Get gets a value in the list[pos].
func (b *Bool) Get(index int) bool {
	data := b.data[8:]

	if index >= b.len {
		panic(fmt.Sprintf("lists.Bool with len %d cannot have position %d set", b.len, index))
	}

	sliceNum := index / 8
	i := binary.Get[uint8](data[sliceNum : sliceNum+1])
	indexInSlice := index - (sliceNum * 8)

	return bits.GetBit(i, uint8(indexInSlice))
}

// Range ranges from "from" (inclusive) to "to" (exclusive). You must read values from
// Range until the returned channel closes or cancel the Context passed. Otherwise
// you will have a goroutine leak.
func (b *Bool) Range(ctx context.Context, from, to int) chan bool {
	if b.len == 0 {
		ch := make(chan bool)
		close(ch)
		return ch
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

	ch := make(chan bool, 1)

	go func() {
		defer close(ch)

		for index := from; index < to; index++ {
			result := b.Get(index)

			select {
			case <-ctx.Done():
				return
			case ch <- result:
			}
		}
	}()

	return ch
}

// Set a boolean in position "pos" to "val".
func (b *Bool) Set(index int, val bool) {
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

func (b *Bool) cap() int {
	return (len(b.data) - 8) * 8 // number of bytes * 8 bit values we can hold
}

// Append appends values to the list of bools.
func (b *Bool) Append(i ...bool) {
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
		addToTotal(b.s, len(b.data)-oldSize)
	}
}

// Slice converts this into a standard []bool. The values aren't linked, so changing
// []bool or calling b.Set(...) will have no affect on the other. If there are no
// entries, this returns a nil slice.
func (b *Bool) Slice() []bool {
	if b.len == 0 {
		return nil
	}
	sl := make([]bool, b.len)

	for i := 0; i < b.len; i++ {
		sl[i] = b.Get(i)
	}
	return sl
}

// Encode returns the []byte to write to output to represent this Bool. If it returns nil,
// no output should be written.
func (b *Bool) Encode() []byte {
	if b.data == nil {
		return nil
	}
	return b.data
}

// Number represents a list of numbers
type Number[I Numbers] struct {
	data        []byte
	sizeInBytes uint8 // 1, 2, 3, 4
	len         int
	isFloat     bool

	s *Struct
}

// NewNumber is used to create a holder for a list of numbers not decoded from an existing []byte stream.
func NewNumber[I Numbers]() *Number[I] {
	var t I

	var n *Number[I]
	var sizeInBytes uint8
	var isFloat bool
	var ft field.Type
	switch any(t).(type) {
	case uint8:
		n = pool.Get(nUint8Pool).(*Number[I])
		sizeInBytes = 1
		ft = field.FTList8
	case uint16:
		n = pool.Get(nUint16Pool).(*Number[I])
		sizeInBytes = 2
		ft = field.FTList16
	case uint32:
		n = pool.Get(nUint32Pool).(*Number[I])
		sizeInBytes = 4
		ft = field.FTList32
	case uint64:
		n = pool.Get(nUint64Pool).(*Number[I])
		sizeInBytes = 8
		ft = field.FTList64
	case int8:
		n = pool.Get(nInt8Pool).(*Number[I])
		sizeInBytes = 1
		ft = field.FTList8
	case int16:
		n = pool.Get(nInt16Pool).(*Number[I])
		sizeInBytes = 2
		ft = field.FTList16
	case int32:
		n = pool.Get(nInt32Pool).(*Number[I])
		sizeInBytes = 4
		ft = field.FTList32
	case int64:
		n = pool.Get(nInt64Pool).(*Number[I])
		sizeInBytes = 8
		ft = field.FTList64
	case float32:
		n = pool.Get(nFloat32Pool).(*Number[I])
		sizeInBytes = 4
		isFloat = true
		ft = field.FTList32
	case float64:
		n = pool.Get(nFloat64Pool).(*Number[I])
		sizeInBytes = 8
		isFloat = true
		ft = field.FTList64
	default:
		panic(fmt.Sprintf("unsupported number type %T", t))
	}
	n.sizeInBytes = sizeInBytes
	n.isFloat = isFloat
	n.data = make([]byte, 8)
	h := GenericHeader(n.data)
	h.SetNext8(uint8(ft))
	return n
}

// NewNumberFromBytes returns a new Number value.
func NewNumberFromBytes[I Numbers](data *[]byte, s *Struct) (*Number[I], error) {
	l := len(*data)
	if l < 8 {
		return nil, fmt.Errorf("header was < 64 bits")
	}

	i := binary.Get[uint64]((*data)[:8])
	items := bits.GetValue[uint64, uint64](i, dataSizeMask, 24)

	var t I

	var n *Number[I]
	var sizeInBytes uint8
	var isFloat bool
	switch any(t).(type) {
	case uint8:
		n = pool.Get(nUint8Pool).(*Number[I])
		sizeInBytes = 1
	case uint16:
		n = pool.Get(nUint16Pool).(*Number[I])
		sizeInBytes = 2
	case uint32:
		n = pool.Get(nUint32Pool).(*Number[I])
		sizeInBytes = 4
	case uint64:
		n = pool.Get(nUint64Pool).(*Number[I])
		sizeInBytes = 8
	case int8:
		n = pool.Get(nInt8Pool).(*Number[I])
		sizeInBytes = 1
	case int16:
		n = pool.Get(nInt16Pool).(*Number[I])
		sizeInBytes = 2
	case int32:
		n = pool.Get(nInt32Pool).(*Number[I])
		sizeInBytes = 4
	case int64:
		n = pool.Get(nInt64Pool).(*Number[I])
		sizeInBytes = 8
	case float32:
		n = pool.Get(nFloat32Pool).(*Number[I])
		sizeInBytes = 4
		isFloat = true
	case float64:
		n = pool.Get(nFloat64Pool).(*Number[I])
		sizeInBytes = 8
		isFloat = true
	default:
		panic(fmt.Sprintf("unsupported number type %T", t))
	}
	n.sizeInBytes = sizeInBytes
	n.isFloat = isFloat

	if items == 0 {
		n.data = (*data)[:8]
		n.len = 0
		n.s = s

		*data = (*data)[8:]
		addToTotal(s, 8)
		return n, nil
	}

	requiredBytes := int(items) * int(n.sizeInBytes)
	requiredWords := requiredBytes / 8
	if requiredBytes%8 != 0 {
		requiredWords++
	}

	if len((*data)[8:]) < int(requiredWords)*8 {
		return nil, fmt.Errorf("malformed: list of numbers[%d bits]: header had data size not consistend with message", sizeInBytes)
	}

	rightBound := (8 * requiredWords) + 8
	sl := (*data)[0:rightBound]
	addToTotal(s, len(sl))

	n.data = sl
	n.len = int(items)
	n.s = s
	*data = (*data)[rightBound:]

	return n, nil
}

// Len returns the number of items in this list.
func (n *Number[I]) Len() int {
	return n.len
}

// Get gets a number stored at the index.
func (n *Number[I]) Get(index int) I {
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

// Range ranges from "from" (inclusive) to "to" (exclusive). You must read values from
// Range until the returned channel closes or cancel the Context passed. Otherwise
// you will have a goroutine leak.
func (n *Number[I]) Range(ctx context.Context, from, to int) chan I {
	if n.len == 0 {
		ch := make(chan I)
		close(ch)
		return ch
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

	ch := make(chan I, 1)

	go func() {
		defer close(ch)

		for index := from; index < to; index++ {
			result := n.Get(index)

			select {
			case <-ctx.Done():
				return
			case ch <- result:
			}
		}
	}()

	return ch
}

func (n *Number[I]) cap() int {
	return len(n.data[8:]) / int(n.sizeInBytes)
}

// Set a number in position "index" to "value".
func (n *Number[I]) Set(index int, value I) {
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
func (n *Number[I]) Append(i ...I) {
	oldSize := len(n.data)
	defer func() {
		updateItems(n.data[:8], n.len)
		if n.s != nil {
			addToTotal(n.s, len(n.data)-oldSize)
		}
	}()

	requiredSize := n.len + len(i)
	// If we have enough internal capacity, then just append inside our capcity.
	if n.cap() >= requiredSize {
		start := n.len
		n.len = n.len + len(i)
		for _, v := range i {
			n.Set(start, v)
		}
		return
	}

	// We don't have enough internal capacity, so let's allocate enough capacity.
	requiredBytes := requiredSize * int(n.sizeInBytes)
	requiredWords := requiredBytes / 8
	if requiredBytes/8 != 0 {
		requiredWords++
	}

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
func (n *Number[I]) Slice() []I {
	if n.len == 0 {
		return nil
	}

	s := make([]I, n.len)
	for v := range n.Range(context.Background(), 0, n.len) {
		s = append(s, v)
	}
	return s
}

// Encode returns the []byte to write to output to represent this Number. If it returns nil,
// no output should be written.
func (n *Number[I]) Encode() []byte {
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
	b := pool.Get(bytesPool).(*Bytes)
	if b.header == nil {
		b.header = NewGenericHeader()
	}
	b.header.SetFirst16(0)
	b.header.SetNext8(uint8(field.FTListBytes))
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
	b := pool.Get(bytesPool).(*Bytes)
	b.header = (*data)[:8]
	*data = (*data)[8:] // Move past the header

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

	addToTotal(s, read) // Add header + data + padding

	b.data = d
	b.s = s
	atomic.StoreInt64(&b.dataSize, int64(read))
	atomic.StoreInt64(&b.padding, int64(paddingNeeded))

	return b, nil
}

// Reset resets all the internal fields to their zero value. Slices are not nilled, but are
// set to their zero size to hold the capacity.
func (b *Bytes) Reset() {
	if b.header != nil {
		b.header.SetFinal40(0)
		b.header.SetFirst16(0)
	}
	if b.data != nil {
		b.data = b.data[0:0]
	}
	b.s = nil
	b.dataSize = 0
	b.padding = 0
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

// Range ranges from "from" (inclusive) to "to" (exclusive). You must read values from
// Range until the returned channel closes or cancel the Context passed. Otherwise
// you will have a goroutine leak. You should NOT modify the returned []byte slice.
func (b *Bytes) Range(ctx context.Context, from, to int) chan []byte {
	if b.Len() == 0 {
		ch := make(chan []byte)
		close(ch)
		return ch
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

	ch := make(chan []byte, 1)

	go func() {
		defer close(ch)

		for index := from; index < to; index++ {
			result := b.Get(index)

			select {
			case <-ctx.Done():
				return
			case ch <- result:
			}
		}
	}()

	return ch
}

// Set a number in position "index" to "value".
func (b *Bytes) Set(index int, value []byte) error {
	if index >= b.Len() {
		panic(fmt.Sprintf("slice out of bounds: index %d in slice of size %d", index, b.Len()))
	}
	if len(value) > math.MaxUint32 {
		return fmt.Errorf("cannot set a value > %dKiB", math.MaxUint32/1024)
	}
	// Record the current size of this value and end padding.  Get new value size and new
	// padding needed. Calculate our new data size.
	oldSize := int64(len(b.data[index]))
	oldPadding := b.padding
	addToTotal(b.s, (oldSize + oldPadding))
	atomic.AddInt64(&b.dataSize, -oldSize)

	b.set(index, value)

	atomic.AddInt64(&b.dataSize, int64(len(value))+4) // data + entry header
	atomic.StoreInt64(&b.padding, PaddingNeeded(b.dataSize))

	// dataSize - oldSize is our data size change.
	// padding - oldPadding is our padding data size change.
	addToTotal(b.s, b.dataSize-oldSize+b.padding-oldPadding)
	b.set(index, value)
	return nil
}

func (b *Bytes) set(index int, value []byte) {
	buff := make([]byte, 4+len(value))
	binary.Put(buff, uint32(len(value)))
	copy(buff[4:], value)
	b.data[index] = buff
}

// Append appends values to the list of []byte.
func (b *Bytes) Append(values ...[]byte) error {
	for _, v := range values {
		if len(v) > math.MaxUint32 {
			return fmt.Errorf("cannot set a value > %dKiB", math.MaxUint32/1024)
		}
	}

	// Record old values
	oldSize := b.dataSize
	oldPadding := b.padding
	oldData := b.data

	newSize := oldSize // We are appending, so our new size starts at the old size

	// Create new slice that can hold our data.
	indexStart := len(b.data)

	b.data = make([][]byte, len(oldData)+len(values))
	copy(b.data, oldData)
	for i, v := range values {
		b.set(indexStart+i, v)
		newSize += int64(len(v)) + 4 // data + entry header
	}
	updateItems(b.header, len(b.data))

	// Record our data size and padding requirements.
	b.dataSize = newSize
	b.padding = PaddingNeeded(newSize)
	if b.s != nil {
		log.Println("b.dataSize: ", b.dataSize)
		log.Println("oldSize: ", oldSize)
		log.Println("b.padding: ", b.padding)
		log.Println("oldPaddin: ", oldPadding)
		addToTotal(b.s, b.dataSize-oldSize+b.padding-oldPadding) // data size + entry header size
	}
	return nil
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
