// Package structs contains objects, functions and methods that are related to reading
// and writing Claw struct types from wire encoding.
package structs

import (
	"fmt"
	"io"
	"math"
	"sync/atomic"
	"unsafe"

	"github.com/bearlytools/claw/internal/binary"
	"github.com/bearlytools/claw/internal/bits"
	"github.com/bearlytools/claw/internal/field"
	"github.com/bearlytools/claw/internal/mapping"
)

const (
	// maxDataSize is the max number that can fit into the dataSize field, which is 40 bits.
	maxDataSize = 1099511627775
	// fieldtype is the struct field type, circular dependency prevents the
	// constant import from claw.go
	fieldType uint8 = 14
)

// Masks to use to pull information from a bitpacked uint64.
var (
	fieldNumMask  = bits.Mask[uint64](0, 16)
	fieldTypeMask = bits.Mask[uint64](16, 24)
	dataSizeMask  = bits.Mask[uint64](24, 64)
)

// GenericHeader is the header of struct.
type GenericHeader []byte

func NewGenericHeader() GenericHeader {
	return GenericHeader(make([]byte, 8))
}

func (g GenericHeader) First16() uint16 {
	return binary.Get[uint16](g[:2])
}

func (g GenericHeader) SetFirst16(u uint16) {
	binary.Put(g[:2], u)
}

func (g GenericHeader) Next8() uint8 {
	return g[2]
}

func (g GenericHeader) SetNext8(u uint8) {
	g[2] = u
}

func (g GenericHeader) Final40() uint64 {
	u := binary.Get[uint64](g)
	return bits.GetValue[uint64, uint64](u, dataSizeMask, 24)
}

func (g GenericHeader) SetFinal40(u uint64) {
	if u > maxDataSize {
		panic(fmt.Sprintf("can't put %d in a 40bit register, max value is 1099511627775", u))
	}
	store := binary.Get[uint64](g)
	store = bits.ClearBits(store, 24, 64)
	store = bits.SetValue(u, store, 24, 64)
	binary.Put(g, store)
}

// structField holds a struct field entry.
type structField struct {
	header GenericHeader
	ptr    unsafe.Pointer
}

// Struct is the basic type for holding a set of values. In claw format, every variable
// must be contained in a Struct.
type Struct struct {
	inList bool

	header GenericHeader
	fields []structField
	excess []byte

	// mapping holds our Mapping object that allows us to understand what field number holds what value type.
	mapping mapping.Map

	parent *Struct

	// structTotal is the total size of this struct in bytes, including header.
	structTotal *int64
}

// New creates a NewStruct that is used to create a *Struct for a specific data type.
func New(fieldNum uint16, dataMap mapping.Map) *Struct {
	// TODO(jdoak): delete?
	/*
		if fieldNum != 0 {
			if parent == nil {
				panic("cannot create a Struct with a fieldNum > 0 and no parent")
			}
			if parent.mapping[fieldNum-1].Type != field.FTStruct {
				panic(fmt.Sprintf("cannot attach a Struct as field %d, that field type is %v", fieldNum, parent.mapping[fieldNum-1].Type))
			}
		}
	*/
	h := GenericHeader(make([]byte, 8))
	h.SetFirst16(fieldNum)
	h.SetNext8(uint8(field.FTStruct))

	s := &Struct{
		header:      h,
		mapping:     dataMap,
		fields:      make([]structField, len(dataMap)),
		structTotal: new(int64),
	}

	/*
		if parent != nil && fieldNum != 0 {
			f := parent.fields[fieldNum-1]
			f.header = h
			f.ptr = unsafe.Pointer(s)
			parent.fields[fieldNum-1] = f
		}
	*/
	addToTotal(s, 8) // the header
	return s
}

// NewFromReader creates a new Struct from data we read in.
func NewFromReader(r io.Reader, maps mapping.Map) (*Struct, error) {
	s := New(0, maps)

	if _, err := s.unmarshal(r); err != nil {
		return nil, err
	}
	return s, nil
}

// IsSet determines if our Struct has a field set or not. If the fieldNum is invalid,
// this simply returns false.
func (s *Struct) IsSet(fieldNum uint16) bool {
	if int(fieldNum) > len(s.mapping) {
		return false
	}
	return s.fields[fieldNum-1].header != nil
}

var boolMask = bits.Mask[uint64](24, 25)

// GetBool gets a bool value from field at fieldNum. This return an error if the field
// is not a bool or fieldNum is not a valid field number. If the field is not set, it
// returns false with no error.
func GetBool(s *Struct, fieldNum uint16) (bool, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return false, err
	}

	f := s.fields[fieldNum-1]
	// Return the zero value of a non-set field.
	if f.header == nil {
		return false, nil
	}

	i := binary.Get[uint64](f.header)
	if bits.GetValue[uint64, uint8](i, boolMask, 24) == 1 {
		return true, nil
	}
	return false, nil
}

func MustGetBool(s *Struct, fieldNum uint16) bool {
	b, err := GetBool(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

// SetBool sets a boolean value in field "fieldNum" to value "value".
func SetBool(s *Struct, fieldNum uint16, value bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		var n uint64
		f.header = GenericHeader(make([]byte, 8))
		n = bits.SetValue(fieldNum, n, 0, 16)
		n = bits.SetValue(uint8(field.FTBool), n, 16, 24)
		if value {
			n = bits.SetBit(n, 24, true)
		}
		binary.Put(f.header, n)
		s.fields[fieldNum-1] = f
		addToTotal(s, 8)
		return nil
	}

	n := binary.Get[uint64](f.header)
	n = bits.SetBit(n, 25, value)
	binary.Put(f.header, n)
	s.fields[fieldNum-1] = f
	return nil
}

func MustSetBool(s *Struct, fieldNum uint16, value bool) {
	err := SetBool(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteBool deletes a boolean and updates our storage total.
func DeleteBool(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBool); err != nil {
		return err
	}
	s.fields[fieldNum-1].header = nil
	return nil
}

// GetNumber gets a number value at fieldNum.
func GetNumber[N Numbers](s *Struct, fieldNum uint16) (N, error) {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return 0, err
	}
	desc := s.mapping[fieldNum-1]

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return 0, fmt.Errorf("error getting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return 0, nil
	}

	if size < 64 {
		b := f.header[3:8]
		if isFloat {
			i := binary.Get[uint32](b)
			return N(math.Float32frombits(uint32(i))), nil
		}
		return N(binary.Get[uint32](b)), nil
	}
	b := *(*[]byte)(f.ptr)
	if isFloat {
		i := binary.Get[uint64](b)
		return N(math.Float64frombits(uint64(i))), nil
	}
	return N(binary.Get[uint64](b)), nil
}

func MustGetNumber[N Numbers](s *Struct, fieldNum uint16) N {
	n, err := GetNumber[N](s, fieldNum)
	if err != nil {
		panic(err)
	}
	return n
}

// SetNumber sets a number value in field "fieldNum" to value "value".
func SetNumber[N Numbers](s *Struct, fieldNum uint16, value N) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping[fieldNum-1]

	size, isFloat, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error setting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum-1]
	// If the field isn't allocated, allocate space.
	if f.header == nil {
		f.header = GenericHeader(make([]byte, 8))
		switch size < 64 {
		case true:
			addToTotal(s, 8)
		case false:
			b := make([]byte, 8)
			f.ptr = unsafe.Pointer(&b)
			addToTotal(s, 16)
		default:
			panic("wtf")
		}
	}

	// Its will store up to 2 uint64s that will be written. 1 is written if we can fit our value
	// in the header, 2 if we can't.
	ints := [2]uint64{}
	// Write our header information.
	ints[0] = bits.SetValue(fieldNum, ints[0], 0, 16)
	ints[0] = bits.SetValue(uint8(desc.Type), ints[0], 16, 24)

	// If the number is a float, convert it to the uint representation.
	if isFloat {
		switch size < 64 {
		case true:
			i := math.Float32bits(float32(value))
			ints[0] = bits.SetValue(i, ints[0], 24, 64)
			binary.Put(f.header[:8], ints[0])
		case false:
			i := math.Float64bits(float64(value))
			ints[1] = i
			binary.Put(f.header[:8], ints[0])
			d := (*[]byte)(f.ptr)
			binary.Put(*d, ints[1])
		default:
			panic("wtf")
		}
	} else {
		// Now encode the Number.
		switch size < 64 {
		case true:
			ints[0] = bits.SetValue(uint32(value), ints[0], 24, 64)
			binary.Put(f.header[:8], ints[0])
		case false:
			ints[1] = uint64(value)
			binary.Put(f.header[:8], ints[0])
			d := (*[]byte)(f.ptr)
			binary.Put(*d, ints[1])
		default:
			panic("wtf")
		}
	}
	s.fields[fieldNum-1] = f
	return nil
}

// DeleteNumber deletes the number and updates our storage total.
func DeleteNumber(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping[fieldNum-1]

	switch desc.Type {
	case field.FTInt8, field.FTInt16, field.FTInt32, field.FTUint8, field.FTUint16, field.FTUint32, field.FTFloat32:
		addToTotal(s, -8)
	case field.FTInt64, field.FTUint64, field.FTFloat64:
		addToTotal(s, -16)
	default:
		panic("wtf")
	}
	f := s.fields[fieldNum-1]
	f.header = nil
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

// GetBytes returns a field of bytes (also our string as well in []byte form). If the value was not
// set, this is returned as nil. If it was set, but empty, this will be []byte{}. It is UNSAFE to modify
// this.
func GetBytes(s *Struct, fieldNum uint16) (*[]byte, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil { // The zero value
		return nil, nil
	}

	if f.ptr == nil { // Set, but value is empty
		return nil, nil
	}

	x := (*[]byte)(f.ptr)
	return x, nil
}

func MustGetBytes(s *Struct, fieldNum uint16) *[]byte {
	b, err := GetBytes(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

// SetBytes sets a field of bytes (also our string as well in []byte form).
func SetBytes(s *Struct, fieldNum uint16, value []byte, isString bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}
	if len(value) == 0 {
		return fmt.Errorf("cannot encode an empty Bytes value")
	}

	if len(value) > maxDataSize {
		return fmt.Errorf("cannot set a String or Byte field to size > 1099511627775")
	}

	f := s.fields[fieldNum-1]
	if value == nil {
		if f.header == nil { // It is already unset
			return nil
		}
		remove := 8
		if f.ptr != nil {
			x := (*[]byte)(f.ptr)
			dataSize := len(*x)
			// Data stored may or may not have padding, so if not aligned this will
			// add the padding.
			remove += SizeWithPadding(dataSize)
		}
		addToTotal(s, -remove)
		f.header = nil
		f.ptr = nil
		s.fields[fieldNum-1] = f
		return nil
	}

	ftype := field.FTBytes
	if isString {
		ftype = field.FTString
	}

	remove := 0
	// If the field isn't allocated, allocate space.
	if f.header == nil {
		f.header = GenericHeader(make([]byte, 8))
	} else { // We need to remove our existing entry size total before applying our new data
		remove += 8 + SizeWithPadding(int(f.header.Final40()))
		addToTotal(s, -remove)
	}
	f.header.SetFirst16(fieldNum)
	f.header.SetNext8(uint8(ftype))
	f.header.SetFinal40(uint64(len(value)))

	f.ptr = unsafe.Pointer(&value)
	// We don't store any padding at this point because we don't want to do another allocation.
	// But we do record the size it would be with padding.
	addToTotal(s, int64(8+SizeWithPadding(len(value))))
	s.fields[fieldNum-1] = f
	return nil
}

func MustSetBytes(s *Struct, fieldNum uint16, value []byte, isString bool) {
	err := SetBytes(s, fieldNum, value, isString)
	if err != nil {
		panic(err)
	}
}

// DeleteBytes deletes a bytes field and updates our storage total.
func DeleteBytes(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTBytes, field.FTString); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil
	}

	f.header = nil
	x := (*[]byte)(f.ptr)
	addToTotal(s, -len(*x))
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

// GetStruct returns a Struct field . If the value was not set, this is returned as nil. If it was set,
// but empty, this will be *Struct with no data.
func GetStruct(s *Struct, fieldNum uint16) (*Struct, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil { // The zero value
		return nil, nil
	}

	x := (*Struct)(f.ptr)
	return x, nil
}

func MustGetStruct(s *Struct, fieldNum uint16) *Struct {
	s, err := GetStruct(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return s
}

// SetStruct sets a Struct field.
func SetStruct(s *Struct, fieldNum uint16, value *Struct) error {
	if s == nil {
		return fmt.Errorf("value cannot be added to a nil Struct")
	}
	if value == nil {
		return fmt.Errorf("value cannot be nil, to delete a Struct use DeleteStruct()")
	}
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return err
	}

	if atomic.LoadInt64(value.structTotal) > maxDataSize {
		return fmt.Errorf("cannot set a Struct field to size > 1099511627775")
	}

	f := s.fields[fieldNum-1]

	value.parent = s
	value.header.SetFirst16(fieldNum)

	var remove int64
	// We need to remove our existing entry size total before applying our new data
	if f.header != nil {
		x := (*Struct)(f.ptr)
		remove += atomic.LoadInt64(x.structTotal)
		x.parent = nil
		addToTotal(s, -remove)
	}

	f.header = value.header

	f.ptr = unsafe.Pointer(value)
	addToTotal(s, atomic.LoadInt64(value.structTotal))
	s.fields[fieldNum-1] = f
	return nil
}

func MustSetStruct(s *Struct, fieldNum uint16, value *Struct) {
	err := SetStruct(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteStruct deletes a Struct field and updates our storage total.
func DeleteStruct(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTStruct); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil
	}

	if f.ptr == nil {
		addToTotal(s, -8)
		return nil
	}

	x := (*Struct)(f.ptr)
	x.parent = nil
	addToTotal(s, -atomic.LoadInt64(x.structTotal))
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

// GetListBool returns a list of bools at fieldNum.
func GetListBool(s *Struct, fieldNum uint16) (*Bool, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBool); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil, nil
	}

	ptr := (*Bool)(f.ptr)

	return ptr, nil
}

func MustGetListBool(s *Struct, fieldNum uint16) *Bool {
	b, err := GetListBool(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListBool(s *Struct, fieldNum uint16, value *Bool) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBool); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header != nil { // We had a previous value stored.
		ptr := (*Bool)(f.ptr)
		addToTotal(s, -len(ptr.data))
	}

	f.header = value.data[:8]
	f.ptr = unsafe.Pointer(value)
	s.fields[fieldNum-1] = f
	value.s = s
	addToTotal(s, len(value.data))
	return nil
}
func MustSetListBool(s *Struct, fieldNum uint16, value *Bool) {
	err := SetListBool(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteListBool deletes a list of bools field and updates our storage total.
func DeleteListBool(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBool); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil
	}

	f.header = nil
	ptr := (*Bool)(f.ptr)
	addToTotal(s, -len(ptr.data))
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

// GetListNumber returns a list of numbers at fieldNum.
func GetListNumber[N Numbers](s *Struct, fieldNum uint16) (*Number[N], error) {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return nil, err
	}
	desc := s.mapping[fieldNum-1]

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil, nil
	}

	_, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return nil, fmt.Errorf("error getting field number %d: %w", fieldNum, err)
	}

	ptr := (*Number[N])(f.ptr)

	return ptr, nil
}

func MustGetListNumber[N Numbers](s *Struct, fieldNum uint16) *Number[N] {
	b, err := GetListNumber[N](s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListNumber[N Numbers](s *Struct, fieldNum uint16, value *Number[N]) error {
	if err := validateFieldNum(fieldNum, s.mapping); err != nil {
		return err
	}
	desc := s.mapping[fieldNum-1]

	_, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error setting field number %d: %w", fieldNum, err)
	}

	f := s.fields[fieldNum-1]
	if f.header != nil { // We had a previous value stored.
		ptr := (*Number[N])(f.ptr)
		addToTotal(s, -len(ptr.data))
	}

	f.header = value.data[:8]
	f.header.SetFirst16(fieldNum)
	f.ptr = unsafe.Pointer(value)
	s.fields[fieldNum-1] = f
	value.s = s
	addToTotal(s, len(value.data))
	return nil
}

func MustSetListNumber[N Numbers](s *Struct, fieldNum uint16, value *Number[N]) {
	err := SetListNumber(s, fieldNum, value)
	if err != nil {
		panic(err)
	}
}

// DeleteListNumber deletes a list of numbers field and updates our storage total.
func DeleteListNumber[N Numbers](s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTList8, field.FTList16, field.FTList32, field.FTList64); err != nil {
		return err
	}
	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil
	}
	desc := s.mapping[fieldNum-1]

	size, _, err := numberToDescCheck[N](desc)
	if err != nil {
		return fmt.Errorf("error deleting field number %d: %w", fieldNum, err)
	}

	ptr := (*Number[N])(f.ptr)
	ptr.s = nil

	reduceBy := int(f.header.Final40()) + int(size) + 8
	addToTotal(s, -reduceBy)

	f.header = nil
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

// GetListStruct returns a list of Structs at fieldNum.
func GetListStruct(s *Struct, fieldNum uint16) (*[]*Struct, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStruct); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil { // The zero value
		return nil, nil
	}

	x := (*[]*Struct)(f.ptr)
	return x, nil
}

func MustGetListStruct(s *Struct, fieldNum uint16) *[]*Struct {
	l, err := GetListStruct(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return l
}

// AppendListStruct adds the values to the list of Structs at fieldNum. Existing items will be retained.
func AppendListStruct(s *Struct, fieldNum uint16, values ...*Struct) error {
	if len(values) == 0 {
		return fmt.Errorf("must add at least a single value")
	}
	if len(values) > maxDataSize {
		return fmt.Errorf("cannot have more than %d items in a list", maxDataSize)
	}
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStruct); err != nil {
		return err
	}

	size := 8 // ListStruct header
	for _, value := range values {
		if atomic.LoadInt64(value.structTotal) > maxDataSize {
			return fmt.Errorf("cannot add a Struct with size > 1099511627775")
		}
		value.header.SetFirst16(fieldNum)
		value.parent = s
		size += int(*value.structTotal)
	}

	f := s.fields[fieldNum-1]

	// If the field isn't allocated, allocate space.
	if f.header == nil {
		f.header = GenericHeader(make([]byte, 8))
		f.header.SetFirst16(fieldNum)
		f.header.SetNext8(uint8(field.FTListStruct))
	}
	f.header.SetFinal40(f.header.Final40() + uint64(len(values)))

	if f.ptr == nil {
		ptr := &values
		f.ptr = unsafe.Pointer(ptr)
	} else {
		ptr := (*[]*Struct)(f.ptr)
		*ptr = append(*ptr, values...)
	}
	addToTotal(s, size)
	s.fields[fieldNum-1] = f

	return nil
}

func MustAppendListStruct(s *Struct, fieldNum uint16, values ...*Struct) {
	err := AppendListStruct(s, fieldNum, values...)
	if err != nil {
		panic(err)
	}
}

// DeleteListStruct deletes a list of Structs field and updates our storage total.
func DeleteListStruct(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListStruct); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil
	}

	size := 8

	x := (*[]*Struct)(f.ptr)
	for _, item := range *x {
		size += int(atomic.LoadInt64(item.structTotal))
	}
	addToTotal(s, -size)
	f.header = nil
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

// GetListBytes returns a list of bytes at fieldNum.
func GetListBytes(s *Struct, fieldNum uint16) (*Bytes, error) {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return nil, err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil, nil
	}

	ptr := (*Bytes)(f.ptr)

	return ptr, nil
}

func MustGetListBytes(s *Struct, fieldNum uint16) *Bytes {
	b, err := GetListBytes(s, fieldNum)
	if err != nil {
		panic(err)
	}
	return b
}

func SetListBytes(s *Struct, fieldNum uint16, value *Bytes) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return err
	}
	value.s = s

	f := s.fields[fieldNum-1]
	if f.header == nil {
		value.header.SetFirst16(fieldNum)
		f.header = value.header
		f.ptr = unsafe.Pointer(value)
		s.fields[fieldNum-1] = f
		addToTotal(s, value.dataSize+value.padding+8)
		return nil
	}
	f.header = value.data[0]
	f.header.SetFirst16(fieldNum)
	ptr := (*Bytes)(f.ptr)
	f.ptr = unsafe.Pointer(value)
	s.fields[fieldNum-1] = f

	addToTotal(s, value.dataSize-ptr.dataSize+value.padding-ptr.padding+8)
	return nil
}

func MustSetListBytes(s *Struct, fieldNum uint16, value *Bytes) {
	if err := SetListBytes(s, fieldNum, value); err != nil {
		panic(err)
	}
}

// DeleteListBytes deletes a list of bytes field and updates our storage total.
func DeleteListBytes(s *Struct, fieldNum uint16) error {
	if err := validateFieldNum(fieldNum, s.mapping, field.FTListBytes); err != nil {
		return err
	}

	f := s.fields[fieldNum-1]
	if f.header == nil {
		return nil
	}

	ptr := (*Bytes)(f.ptr)
	addToTotal(s, -(ptr.dataSize + ptr.padding))

	f.header = nil
	f.ptr = nil
	s.fields[fieldNum-1] = f
	return nil
}

func addToTotal[N int64 | int | uint | uint64](s *Struct, value N) {
	v := atomic.AddInt64(s.structTotal, int64(value))
	s.header.SetFinal40(uint64(v))
	var ptr = s.parent
	for {
		if ptr == nil {
			return
		}
		v := atomic.AddInt64(ptr.structTotal, int64(value))
		ptr.header.SetFinal40(uint64(v))
		ptr = ptr.parent
	}
}

// validateFieldNum will validate that the fieldNum is > 0, that the type is described in the mapping.Map,
// and if len(ftypes) != 0, that the ftype and mapping.Map[fieldNum-1].Type are the same.
func validateFieldNum(fieldNum uint16, maps mapping.Map, ftypes ...field.Type) error {
	if fieldNum == 0 {
		return fmt.Errorf("fieldNum cannot be 0")
	}
	if int(fieldNum) > len(maps) {
		return fmt.Errorf("fieldNum is > the number of possible fields")
	}
	if len(ftypes) == 0 {
		return nil
	}

	desc := maps[fieldNum-1]
	found := false
	for _, ftype := range ftypes {
		if desc.Type == ftype {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("fieldNum(%d) was %v, which was not valid", fieldNum, desc.Type)
	}
	return nil
}

func numberToDescCheck[N Numbers](desc *mapping.FieldDesc) (size uint8, isFloat bool, err error) {
	var t N
	switch any(t).(type) {
	case uint8:
		switch desc.Type {
		case field.FTUint8:
		case field.FTList8:
			if desc.ListType != field.FTUint8 {
				return 0, false, fmt.Errorf("fieldNum is not a []uint8 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint8 or []uint8 type, was %v", desc.Type)
		}
		size = 8
	case uint16:
		switch desc.Type {
		case field.FTUint16:
		case field.FTList16:
			if desc.ListType != field.FTUint16 {
				return 0, false, fmt.Errorf("fieldNum is not a []uint16 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint16 or []uint16 type, was %v", desc.Type)
		}
		size = 16
	case uint32:
		switch desc.Type {
		case field.FTUint32:
		case field.FTList32:
			if desc.ListType != field.FTUint32 {
				return 0, false, fmt.Errorf("fieldNum is not a []uint32 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint32 or []uint32 type, was %v", desc.Type)
		}
		size = 32
	case uint64:
		switch desc.Type {
		case field.FTUint64:
		case field.FTList64:
			if desc.ListType != field.FTUint64 {
				return 0, false, fmt.Errorf("fieldNum is not a []uint64 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a uint64 or []uint64 type, was %v", desc.Type)
		}
		size = 64
	case int8:
		switch desc.Type {
		case field.FTInt8:
		case field.FTList8:
			if desc.ListType != field.FTInt8 {
				return 0, false, fmt.Errorf("fieldNum is not a []int8 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int8 or []int8 type, was %v", desc.Type)
		}
		size = 8
	case int16:
		switch desc.Type {
		case field.FTInt16:
		case field.FTList16:
			if desc.ListType != field.FTInt16 {
				return 0, false, fmt.Errorf("fieldNum is not a []int16 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int16 or []int16 type, was %v", desc.Type)
		}
		size = 16
	case int32:
		switch desc.Type {
		case field.FTInt32:
		case field.FTList32:
			if desc.ListType != field.FTInt32 {
				return 0, false, fmt.Errorf("fieldNum is not a []int32 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int32 or []int32 type, was %v", desc.Type)
		}
		size = 32
	case int64:
		switch desc.Type {
		case field.FTInt64:
		case field.FTList64:
			if desc.ListType != field.FTInt8 {
				return 0, false, fmt.Errorf("fieldNum is not a []int64 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a int64 or []int64 type, was %v", desc.Type)
		}
		size = 64
	case float32:
		switch desc.Type {
		case field.FTFloat32:
		case field.FTList32:
			if desc.ListType != field.FTFloat32 {
				return 0, false, fmt.Errorf("fieldNum is not a []float32 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a float32 or []float32 type, was %v", desc.Type)
		}
		size = 32
		isFloat = true
	case float64:
		switch desc.Type {
		case field.FTFloat64:
		case field.FTList64:
			if desc.ListType != field.FTFloat32 {
				return 0, false, fmt.Errorf("fieldNum is not a []float64 type, was []%v", desc.ListType)
			}
		default:
			return 0, false, fmt.Errorf("fieldNum is not a float64 or []float64 type, was %v", desc.Type)
		}
		size = 64
		isFloat = true
	default:
		return 0, false, fmt.Errorf("passed a number value of %T that we do not support", t)
	}
	return size, isFloat, nil
}
