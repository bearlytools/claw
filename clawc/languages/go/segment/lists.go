package segment

import (
	"encoding/binary"
	"iter"
	"math"
	"reflect"
	"unsafe"

	"github.com/bearlytools/claw/clawc/languages/go/field"
	"github.com/bearlytools/claw/clawc/languages/go/mapping"
)

// Bools represents a list of boolean values with external buffer storage.
type Bools struct {
	parent   *Struct
	fieldNum uint16
	items    []bool
	dirty    bool
}

// NewBools creates a new Bools list attached to a struct field.
func NewBools(parent *Struct, fieldNum uint16) *Bools {
	b := &Bools{
		parent:   parent,
		fieldNum: fieldNum,
		items:    make([]bool, 0, 8),
	}
	parent.RegisterDirtyList(fieldNum, b)
	return b
}

// Len returns the number of items.
func (b *Bools) Len() int {
	return len(b.items)
}

// Append adds booleans to the list.
func (b *Bools) Append(values ...bool) {
	b.items = append(b.items, values...)
	b.dirty = true
}

// Get returns the boolean at index.
func (b *Bools) Get(index int) bool {
	if index < 0 || index >= len(b.items) {
		panic("segment: list index out of range")
	}
	return b.items[index]
}

// Set sets the boolean at index.
func (b *Bools) Set(index int, value bool) {
	if index < 0 || index >= len(b.items) {
		panic("segment: list index out of range")
	}
	b.items[index] = value
	b.dirty = true
}

// SetAll replaces all items in the list.
func (b *Bools) SetAll(values []bool) {
	b.items = make([]bool, len(values))
	copy(b.items, values)
	b.dirty = true
}

// All returns an iterator over all boolean values.
func (b *Bools) All() iter.Seq[bool] {
	return func(yield func(bool) bool) {
		for _, v := range b.items {
			if !yield(v) {
				return
			}
		}
	}
}

// Range returns an iterator from index 'from' (inclusive) to 'to' (exclusive).
func (b *Bools) Range(from, to int) iter.Seq[bool] {
	return func(yield func(bool) bool) {
		if from < 0 {
			from = 0
		}
		if to > len(b.items) {
			to = len(b.items)
		}
		for i := from; i < to; i++ {
			if !yield(b.items[i]) {
				return
			}
		}
	}
}

// Slice returns a copy of all items.
func (b *Bools) Slice() []bool {
	result := make([]bool, len(b.items))
	copy(result, b.items)
	return result
}

// SyncToSegment syncs the list data to the parent struct's segment.
func (b *Bools) SyncToSegment() error {
	if !b.dirty {
		return nil
	}

	if len(b.items) == 0 {
		b.parent.removeField(b.fieldNum)
		b.dirty = false
		return nil
	}

	// Bools are packed as bits, 8 per byte
	numBytes := (len(b.items) + 7) / 8
	padding := paddingNeeded(numBytes)
	totalSize := HeaderSize + numBytes + padding

	fieldData := make([]byte, totalSize)
	// Write header with total size (not item count) so parseFieldIndex can skip correctly
	EncodeHeader(fieldData[0:8], b.fieldNum, field.FTListBools, uint64(totalSize))

	// Pack bools as bits
	for i, v := range b.items {
		if v {
			byteIdx := 8 + i/8
			bitIdx := uint(i % 8)
			fieldData[byteIdx] |= 1 << bitIdx
		}
	}

	b.parent.insertField(b.fieldNum, fieldData)
	b.dirty = false
	return nil
}

// Numbers represents a list of numeric values with external buffer storage.
// During construction, values are appended to an external buffer.
// Before Marshal, the list is synced to the parent struct's segment.
type Numbers[I Number] struct {
	parent   *Struct
	fieldNum uint16
	header   []byte // 8-byte header
	data     []byte // External buffer: header + data
	len      int    // Number of items
	dirty    bool   // True if data differs from segment
}

// NewNumbers creates a new Numbers list attached to a struct field.
func NewNumbers[I Number](parent *Struct, fieldNum uint16) *Numbers[I] {
	n := &Numbers[I]{
		parent:   parent,
		fieldNum: fieldNum,
		header:   make([]byte, HeaderSize),
		data:     make([]byte, 0, 64), // Initial capacity
	}

	// Determine field type from I using reflection kind
	var zero I
	var ft field.Type
	switch reflect.TypeOf(zero).Kind() {
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

	EncodeHeader(n.header, fieldNum, ft, 0)

	// Register for syncing before Marshal
	parent.RegisterDirtyList(fieldNum, n)

	return n
}

// itemSize returns the size in bytes of one item.
func (n *Numbers[I]) itemSize() int {
	var zero I
	return int(unsafe.Sizeof(zero))
}

// Len returns the number of items in the list.
func (n *Numbers[I]) Len() int {
	return n.len
}

// Append adds values to the list.
func (n *Numbers[I]) Append(values ...I) {
	itemSz := n.itemSize()

	// Ensure capacity
	needed := len(values) * itemSz
	if cap(n.data)-len(n.data) < needed {
		newCap := cap(n.data) * 2
		if newCap < len(n.data)+needed {
			newCap = len(n.data) + needed
		}
		newData := make([]byte, len(n.data), newCap)
		copy(newData, n.data)
		n.data = newData
	}

	// Get the underlying kind for type switching
	var zero I
	kind := reflect.TypeOf(zero).Kind()

	// Append each value
	for _, v := range values {
		offset := len(n.data)
		n.data = n.data[:offset+itemSz]

		rv := reflect.ValueOf(v)
		switch kind {
		case reflect.Int8:
			n.data[offset] = byte(rv.Int())
		case reflect.Uint8:
			n.data[offset] = byte(rv.Uint())
		case reflect.Int16:
			binary.LittleEndian.PutUint16(n.data[offset:], uint16(rv.Int()))
		case reflect.Uint16:
			binary.LittleEndian.PutUint16(n.data[offset:], uint16(rv.Uint()))
		case reflect.Int32:
			binary.LittleEndian.PutUint32(n.data[offset:], uint32(rv.Int()))
		case reflect.Uint32:
			binary.LittleEndian.PutUint32(n.data[offset:], uint32(rv.Uint()))
		case reflect.Float32:
			binary.LittleEndian.PutUint32(n.data[offset:], math.Float32bits(float32(rv.Float())))
		case reflect.Int64:
			binary.LittleEndian.PutUint64(n.data[offset:], uint64(rv.Int()))
		case reflect.Uint64:
			binary.LittleEndian.PutUint64(n.data[offset:], rv.Uint())
		case reflect.Float64:
			binary.LittleEndian.PutUint64(n.data[offset:], math.Float64bits(rv.Float()))
		}
	}

	n.len += len(values)
	n.dirty = true
}

// Get returns the value at index.
func (n *Numbers[I]) Get(index int) I {
	if index < 0 || index >= n.len {
		panic("segment: list index out of range")
	}

	itemSz := n.itemSize()
	offset := index * itemSz

	var zero I
	kind := reflect.TypeOf(zero).Kind()
	result := reflect.New(reflect.TypeOf(zero)).Elem()

	switch kind {
	case reflect.Int8:
		result.SetInt(int64(int8(n.data[offset])))
	case reflect.Uint8:
		result.SetUint(uint64(n.data[offset]))
	case reflect.Int16:
		result.SetInt(int64(int16(binary.LittleEndian.Uint16(n.data[offset:]))))
	case reflect.Uint16:
		result.SetUint(uint64(binary.LittleEndian.Uint16(n.data[offset:])))
	case reflect.Int32:
		result.SetInt(int64(int32(binary.LittleEndian.Uint32(n.data[offset:]))))
	case reflect.Uint32:
		result.SetUint(uint64(binary.LittleEndian.Uint32(n.data[offset:])))
	case reflect.Float32:
		result.SetFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(n.data[offset:]))))
	case reflect.Int64:
		result.SetInt(int64(binary.LittleEndian.Uint64(n.data[offset:])))
	case reflect.Uint64:
		result.SetUint(binary.LittleEndian.Uint64(n.data[offset:]))
	case reflect.Float64:
		result.SetFloat(math.Float64frombits(binary.LittleEndian.Uint64(n.data[offset:])))
	default:
		return zero
	}

	return result.Interface().(I)
}

// Set sets the value at index.
func (n *Numbers[I]) Set(index int, value I) {
	if index < 0 || index >= n.len {
		panic("segment: list index out of range")
	}

	itemSz := n.itemSize()
	offset := index * itemSz

	var zero I
	kind := reflect.TypeOf(zero).Kind()
	rv := reflect.ValueOf(value)

	switch kind {
	case reflect.Int8:
		n.data[offset] = byte(rv.Int())
	case reflect.Uint8:
		n.data[offset] = byte(rv.Uint())
	case reflect.Int16:
		binary.LittleEndian.PutUint16(n.data[offset:], uint16(rv.Int()))
	case reflect.Uint16:
		binary.LittleEndian.PutUint16(n.data[offset:], uint16(rv.Uint()))
	case reflect.Int32:
		binary.LittleEndian.PutUint32(n.data[offset:], uint32(rv.Int()))
	case reflect.Uint32:
		binary.LittleEndian.PutUint32(n.data[offset:], uint32(rv.Uint()))
	case reflect.Float32:
		binary.LittleEndian.PutUint32(n.data[offset:], math.Float32bits(float32(rv.Float())))
	case reflect.Int64:
		binary.LittleEndian.PutUint64(n.data[offset:], uint64(rv.Int()))
	case reflect.Uint64:
		binary.LittleEndian.PutUint64(n.data[offset:], rv.Uint())
	case reflect.Float64:
		binary.LittleEndian.PutUint64(n.data[offset:], math.Float64bits(rv.Float()))
	}
	n.dirty = true
}

// SetAll replaces all items in the list.
func (n *Numbers[I]) SetAll(values []I) {
	itemSz := n.itemSize()
	totalSize := len(values) * itemSz

	n.data = make([]byte, totalSize)
	n.len = len(values)

	var zero I
	kind := reflect.TypeOf(zero).Kind()

	for i, v := range values {
		offset := i * itemSz
		rv := reflect.ValueOf(v)
		switch kind {
		case reflect.Int8:
			n.data[offset] = byte(rv.Int())
		case reflect.Uint8:
			n.data[offset] = byte(rv.Uint())
		case reflect.Int16:
			binary.LittleEndian.PutUint16(n.data[offset:], uint16(rv.Int()))
		case reflect.Uint16:
			binary.LittleEndian.PutUint16(n.data[offset:], uint16(rv.Uint()))
		case reflect.Int32:
			binary.LittleEndian.PutUint32(n.data[offset:], uint32(rv.Int()))
		case reflect.Uint32:
			binary.LittleEndian.PutUint32(n.data[offset:], uint32(rv.Uint()))
		case reflect.Float32:
			binary.LittleEndian.PutUint32(n.data[offset:], math.Float32bits(float32(rv.Float())))
		case reflect.Int64:
			binary.LittleEndian.PutUint64(n.data[offset:], uint64(rv.Int()))
		case reflect.Uint64:
			binary.LittleEndian.PutUint64(n.data[offset:], rv.Uint())
		case reflect.Float64:
			binary.LittleEndian.PutUint64(n.data[offset:], math.Float64bits(rv.Float()))
		}
	}
	n.dirty = true
}

// All returns an iterator over all values.
func (n *Numbers[I]) All() iter.Seq[I] {
	return func(yield func(I) bool) {
		for i := 0; i < n.len; i++ {
			if !yield(n.Get(i)) {
				return
			}
		}
	}
}

// Range returns an iterator from index 'from' (inclusive) to 'to' (exclusive).
func (n *Numbers[I]) Range(from, to int) iter.Seq[I] {
	return func(yield func(I) bool) {
		if from < 0 {
			from = 0
		}
		if to > n.len {
			to = n.len
		}
		for i := from; i < to; i++ {
			if !yield(n.Get(i)) {
				return
			}
		}
	}
}

// Slice returns a copy of all items.
func (n *Numbers[I]) Slice() []I {
	result := make([]I, n.len)
	for i := 0; i < n.len; i++ {
		result[i] = n.Get(i)
	}
	return result
}

// SyncToSegment syncs the list data to the parent struct's segment.
func (n *Numbers[I]) SyncToSegment() error {
	if !n.dirty {
		return nil
	}

	if n.len == 0 {
		n.parent.removeField(n.fieldNum)
		n.dirty = false
		return nil
	}

	// Calculate total size with padding
	dataLen := len(n.data)
	padding := paddingNeeded(dataLen)
	totalSize := HeaderSize + dataLen + padding

	// Create field data
	fieldData := make([]byte, totalSize)

	// Write header with total size (not item count) so parseFieldIndex can skip correctly
	EncodeHeader(fieldData[0:8], n.fieldNum, n.fieldType(), uint64(totalSize))

	// Copy data
	copy(fieldData[8:8+dataLen], n.data)

	// Insert into parent segment
	n.parent.insertField(n.fieldNum, fieldData)
	n.dirty = false

	return nil
}

// fieldType returns the field type for this Numbers list.
func (n *Numbers[I]) fieldType() field.Type {
	var zero I
	switch reflect.TypeOf(zero).Kind() {
	case reflect.Int8:
		return field.FTListInt8
	case reflect.Int16:
		return field.FTListInt16
	case reflect.Int32:
		return field.FTListInt32
	case reflect.Int64:
		return field.FTListInt64
	case reflect.Uint8:
		return field.FTListUint8
	case reflect.Uint16:
		return field.FTListUint16
	case reflect.Uint32:
		return field.FTListUint32
	case reflect.Uint64:
		return field.FTListUint64
	case reflect.Float32:
		return field.FTListFloat32
	case reflect.Float64:
		return field.FTListFloat64
	default:
		return 0
	}
}

// Strings represents a list of strings with external buffer storage.
type Strings struct {
	parent   *Struct
	fieldNum uint16
	items    []string // Store strings directly
	dirty    bool
}

// NewStrings creates a new Strings list attached to a struct field.
func NewStrings(parent *Struct, fieldNum uint16) *Strings {
	s := &Strings{
		parent:   parent,
		fieldNum: fieldNum,
		items:    make([]string, 0, 8),
	}
	parent.RegisterDirtyList(fieldNum, s)
	return s
}

// Len returns the number of items.
func (s *Strings) Len() int {
	return len(s.items)
}

// Append adds strings to the list.
func (s *Strings) Append(values ...string) {
	s.items = append(s.items, values...)
	s.dirty = true
}

// Get returns the string at index.
func (s *Strings) Get(index int) string {
	if index < 0 || index >= len(s.items) {
		panic("segment: list index out of range")
	}
	return s.items[index]
}

// Set sets the string at index.
func (s *Strings) Set(index int, value string) {
	if index < 0 || index >= len(s.items) {
		panic("segment: list index out of range")
	}
	s.items[index] = value
	s.dirty = true
}

// SetAll replaces all items in the list.
func (s *Strings) SetAll(values []string) {
	s.items = make([]string, len(values))
	copy(s.items, values)
	s.dirty = true
}

// All returns an iterator over all strings.
func (s *Strings) All() iter.Seq[string] {
	return func(yield func(string) bool) {
		for _, v := range s.items {
			if !yield(v) {
				return
			}
		}
	}
}

// Range returns an iterator from index 'from' (inclusive) to 'to' (exclusive).
func (s *Strings) Range(from, to int) iter.Seq[string] {
	return func(yield func(string) bool) {
		if from < 0 {
			from = 0
		}
		if to > len(s.items) {
			to = len(s.items)
		}
		for i := from; i < to; i++ {
			if !yield(s.items[i]) {
				return
			}
		}
	}
}

// Slice returns a copy of all items.
func (s *Strings) Slice() []string {
	result := make([]string, len(s.items))
	copy(result, s.items)
	return result
}

// SyncToSegment syncs the list data to the parent struct's segment.
func (s *Strings) SyncToSegment() error {
	if !s.dirty {
		return nil
	}

	if len(s.items) == 0 {
		s.parent.removeField(s.fieldNum)
		s.dirty = false
		return nil
	}

	// Calculate total size: header + (4-byte length + data) for each item + padding
	dataSize := 0
	for _, item := range s.items {
		dataSize += 4 + len(item) // 4-byte length header + data
	}
	padding := paddingNeeded(dataSize)
	totalSize := HeaderSize + dataSize + padding

	// Create field data
	fieldData := make([]byte, totalSize)

	// Write header with total size (not item count) so parseFieldIndex can skip correctly
	EncodeHeader(fieldData[0:8], s.fieldNum, field.FTListStrings, uint64(totalSize))

	// Write each item
	offset := 8
	for _, item := range s.items {
		// Write 4-byte length
		binary.LittleEndian.PutUint32(fieldData[offset:], uint32(len(item)))
		offset += 4

		// Write data
		copy(fieldData[offset:], item)
		offset += len(item)
	}

	// Insert into parent segment
	s.parent.insertField(s.fieldNum, fieldData)
	s.dirty = false

	return nil
}

// Bytes represents a list of byte slices with external buffer storage.
type Bytes struct {
	parent   *Struct
	fieldNum uint16
	items    [][]byte
	dirty    bool
}

// NewBytes creates a new Bytes list attached to a struct field.
func NewBytes(parent *Struct, fieldNum uint16) *Bytes {
	b := &Bytes{
		parent:   parent,
		fieldNum: fieldNum,
		items:    make([][]byte, 0, 8),
	}
	parent.RegisterDirtyList(fieldNum, b)
	return b
}

// Len returns the number of items.
func (b *Bytes) Len() int {
	return len(b.items)
}

// Append adds byte slices to the list.
func (b *Bytes) Append(values ...[]byte) {
	b.items = append(b.items, values...)
	b.dirty = true
}

// Get returns the bytes at index.
func (b *Bytes) Get(index int) []byte {
	if index < 0 || index >= len(b.items) {
		panic("segment: list index out of range")
	}
	return b.items[index]
}

// Set sets the bytes at index.
func (b *Bytes) Set(index int, value []byte) {
	if index < 0 || index >= len(b.items) {
		panic("segment: list index out of range")
	}
	b.items[index] = make([]byte, len(value))
	copy(b.items[index], value)
	b.dirty = true
}

// SetAll replaces all items in the list.
func (b *Bytes) SetAll(values [][]byte) {
	b.items = make([][]byte, len(values))
	for i, v := range values {
		b.items[i] = make([]byte, len(v))
		copy(b.items[i], v)
	}
	b.dirty = true
}

// All returns an iterator over all byte slices.
func (b *Bytes) All() iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		for _, v := range b.items {
			if !yield(v) {
				return
			}
		}
	}
}

// Range returns an iterator from index 'from' (inclusive) to 'to' (exclusive).
func (b *Bytes) Range(from, to int) iter.Seq[[]byte] {
	return func(yield func([]byte) bool) {
		if from < 0 {
			from = 0
		}
		if to > len(b.items) {
			to = len(b.items)
		}
		for i := from; i < to; i++ {
			if !yield(b.items[i]) {
				return
			}
		}
	}
}

// Slice returns a copy of all items.
func (b *Bytes) Slice() [][]byte {
	result := make([][]byte, len(b.items))
	for i, v := range b.items {
		result[i] = make([]byte, len(v))
		copy(result[i], v)
	}
	return result
}

// SyncToSegment syncs the list data to the parent struct's segment.
func (b *Bytes) SyncToSegment() error {
	if !b.dirty {
		return nil
	}

	if len(b.items) == 0 {
		b.parent.removeField(b.fieldNum)
		b.dirty = false
		return nil
	}

	// Calculate total size
	dataSize := 0
	for _, item := range b.items {
		dataSize += 4 + len(item)
	}
	padding := paddingNeeded(dataSize)
	totalSize := HeaderSize + dataSize + padding

	// Create field data
	fieldData := make([]byte, totalSize)

	// Write header with total size (not item count) so parseFieldIndex can skip correctly
	EncodeHeader(fieldData[0:8], b.fieldNum, field.FTListBytes, uint64(totalSize))

	// Write each item
	offset := 8
	for _, item := range b.items {
		binary.LittleEndian.PutUint32(fieldData[offset:], uint32(len(item)))
		offset += 4
		copy(fieldData[offset:], item)
		offset += len(item)
	}

	// Insert into parent segment
	b.parent.insertField(b.fieldNum, fieldData)
	b.dirty = false

	return nil
}

// Structs represents a list of structs with external storage.
type Structs struct {
	parent   *Struct
	fieldNum uint16
	mapping  *mapping.Map
	items    []*Struct
	dirty    bool
}

// NewStructs creates a new Structs list attached to a struct field.
// If the field already exists in the parent segment, it parses the items from the data.
func NewStructs(parent *Struct, fieldNum uint16, m *mapping.Map) *Structs {
	s := &Structs{
		parent:   parent,
		fieldNum: fieldNum,
		mapping:  m,
		items:    make([]*Struct, 0, 8),
	}

	// If the field exists in the segment, parse it
	offset, size := parent.FieldOffset(fieldNum)
	if size > HeaderSize {
		s.parseFromSegment(parent.seg.data[offset : offset+size])
	}

	parent.RegisterDirtyList(fieldNum, s)
	return s
}

// parseFromSegment parses struct items from segment data.
func (s *Structs) parseFromSegment(data []byte) {
	if len(data) < HeaderSize {
		return
	}

	// Skip the list header
	pos := HeaderSize
	for pos+HeaderSize <= len(data) {
		// Decode the struct header to get its size
		_, fieldType, final40 := DecodeHeader(data[pos : pos+HeaderSize])

		if fieldType != field.FTStruct {
			// Not a struct, skip
			break
		}

		structSize := int(final40)
		if structSize < HeaderSize || pos+structSize > len(data) {
			break
		}

		// Create a struct viewing this portion of the data
		item := &Struct{
			seg:        &Segment{data: data[pos : pos+structSize]},
			mapping:    s.mapping,
			fieldIndex: make([]fieldEntry, len(s.mapping.Fields)),
		}
		parseFieldIndex(item)
		s.items = append(s.items, item)

		pos += structSize
	}
}

// Len returns the number of items.
func (s *Structs) Len() int {
	return len(s.items)
}

// Append adds structs to the list.
func (s *Structs) Append(values ...*Struct) {
	s.items = append(s.items, values...)
	s.dirty = true
}

// Get returns the struct at index.
func (s *Structs) Get(index int) *Struct {
	if index < 0 || index >= len(s.items) {
		panic("segment: list index out of range")
	}
	return s.items[index]
}

// NewItem creates a new struct item with the list's mapping.
func (s *Structs) NewItem() *Struct {
	return New(s.mapping)
}

// Set sets the struct at index.
func (s *Structs) Set(index int, value *Struct) {
	if index < 0 || index >= len(s.items) {
		panic("segment: list index out of range")
	}
	s.items[index] = value
	s.dirty = true
}

// SetAll replaces all items in the list.
func (s *Structs) SetAll(values []*Struct) {
	s.items = make([]*Struct, len(values))
	copy(s.items, values)
	s.dirty = true
}

// All returns an iterator over all structs.
func (s *Structs) All() iter.Seq[*Struct] {
	return func(yield func(*Struct) bool) {
		for _, v := range s.items {
			if !yield(v) {
				return
			}
		}
	}
}

// Range returns an iterator from index 'from' (inclusive) to 'to' (exclusive).
func (s *Structs) Range(from, to int) iter.Seq[*Struct] {
	return func(yield func(*Struct) bool) {
		if from < 0 {
			from = 0
		}
		if to > len(s.items) {
			to = len(s.items)
		}
		for i := from; i < to; i++ {
			if !yield(s.items[i]) {
				return
			}
		}
	}
}

// Slice returns a copy of all items (shallow copy of pointers).
func (s *Structs) Slice() []*Struct {
	result := make([]*Struct, len(s.items))
	copy(result, s.items)
	return result
}

// SyncToSegment syncs the list data to the parent struct's segment.
func (s *Structs) SyncToSegment() error {
	if !s.dirty {
		return nil
	}

	if len(s.items) == 0 {
		s.parent.removeField(s.fieldNum)
		s.dirty = false
		return nil
	}

	// First sync all items' dirty lists recursively.
	// This ensures nested structs have their data written before we copy.
	for _, item := range s.items {
		if err := item.syncDirtyLists(); err != nil {
			return err
		}
	}

	// Calculate total size: sum of all struct sizes
	dataSize := 0
	for _, item := range s.items {
		dataSize += item.seg.Len()
	}
	totalSize := HeaderSize + dataSize

	// Create field data
	fieldData := make([]byte, totalSize)

	// Write header with total size (not item count) so parseFieldIndex can skip correctly
	EncodeHeader(fieldData[0:8], s.fieldNum, field.FTListStructs, uint64(totalSize))

	// Write each struct's segment data
	offset := 8
	for i, item := range s.items {
		// Update the struct's field number to be its index in the list
		EncodeHeaderFieldNum(item.seg.data[0:2], uint16(i))

		// Copy the struct's segment data
		copy(fieldData[offset:], item.seg.data)
		offset += item.seg.Len()
	}

	// Insert into parent segment
	s.parent.insertField(s.fieldNum, fieldData)
	s.dirty = false

	return nil
}
